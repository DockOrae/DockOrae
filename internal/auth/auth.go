package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/argon2"
)

const TOKEN_TTL = 7 * 24 * 3600 // 秒

// ---------------- 密码 (argon2id PHC 字符串) ----------------

// HashPassword 生成与旧版 argon2 crate 默认参数兼容的 PHC 字符串
// (Argon2id v19, m=19456 KiB, t=2, p=1, 32 字节输出)
func HashPassword(pw string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(pw), salt, 2, 19456, 1, 32)
	return fmt.Sprintf("$argon2id$v=19$m=19456,t=2,p=1$%s$%s",
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash)), nil
}

// VerifyPassword 校验任意参数的 argon2id PHC 字符串(兼容旧版生成的哈希)
func VerifyPassword(pw, encoded string) bool {
	parts := strings.Split(encoded, "$")
	// $argon2id$v=19$m=...,t=...,p=...$salt$hash
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	var mem, timev, threads int
	for _, kv := range strings.Split(parts[3], ",") {
		key, val, _ := strings.Cut(kv, "=")
		n, err := strconv.Atoi(val)
		if err != nil {
			return false
		}
		switch key {
		case "m":
			mem = n
		case "t":
			timev = n
		case "p":
			threads = n
		}
	}
	if mem <= 0 || timev <= 0 || threads <= 0 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(pw), salt, uint32(timev), uint32(mem), uint8(threads), uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// ---------------- JWT (HS256, 与旧版同格式) ----------------

func b64(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

// MakeToken 签发 JWT(HS256)。pca 为用户的 password_changed_at(安全凭据变更时间戳):
// 修改密码/启用或关闭 2FA 后 pca 递增,旧 token(pca 落后)在中间件校验时被拒绝(SEC-003)。
func MakeToken(secret, username string, ttlSeconds, pca int64) string {
	now := time.Now().Unix()
	header := b64([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(map[string]any{
		"sub": username, "iat": now, "exp": now + ttlSeconds, "pca": pca,
	})
	payload := b64(payloadBytes)
	signing := header + "." + payload
	return signing + "." + sign(secret, signing)
}

// VerifyToken 校验 JWT,返回 (username, pca, ok)。
// 旧格式 token(无 pca 字段,签发于引入本机制之前)按 pca=0 处理——与用户 pca=0
// (从未变更过安全凭据)匹配,保证升级后已有会话不失效。
func VerifyToken(secret, token string) (string, int64, bool) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return "", 0, false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(mac.Sum(nil), sig) {
		return "", 0, false
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", 0, false
	}
	var payload struct {
		Sub string `json:"sub"`
		Exp int64  `json:"exp"`
		Pca int64  `json:"pca"`
	}
	if json.Unmarshal(payloadBytes, &payload) != nil {
		return "", 0, false
	}
	if payload.Exp < time.Now().Unix() {
		return "", 0, false
	}
	return payload.Sub, payload.Pca, true
}

func sign(secret, data string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return b64(mac.Sum(nil))
}

// ---------------- TOTP (RFC 6238) ----------------

// GenerateTotpSecret 生成 20 字节随机数 → base32(32 字符,无填充),与旧版一致
func GenerateTotpSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base32NoPad(buf), nil
}

func base32NoPad(b []byte) string {
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
}

func TotpURI(secret, issuer, account string) string {
	iss := url.PathEscape(issuer)
	acc := url.PathEscape(account)
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		iss, acc, secret, iss)
}

// VerifyTotp 校验 6 位动态码(允许 ±1 个时间窗口的时钟偏移)
func VerifyTotp(secret, code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	for _, r := range code {
		if r < '0' || r > '9' {
			return false
		}
	}
	ok, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return err == nil && ok
}

// ---------------- Token 提取(供 api 中间件使用) ----------------

func BearerToken(h string) string {
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}

func QueryToken(q url.Values) string {
	t := q.Get("token")
	if t != "" {
		return t
	}
	for _, v := range q["token"] {
		if v != "" {
			return v
		}
	}
	return ""
}
