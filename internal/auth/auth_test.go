package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTokenPCA(t *testing.T) {
	secret := "test-secret-123"
	// 携带 pca 签发 → 原样返回
	tok := MakeToken(secret, "admin", 3600, 12345)
	user, pca, ok := VerifyToken(secret, tok)
	if !ok || user != "admin" || pca != 12345 {
		t.Fatalf("VerifyToken = (%q, %d, %v), want (admin, 12345, true)", user, pca, ok)
	}
	// 伪造的 pca 无法通过签名校验(篡改 payload 后签名失效)
	parts := strings.Split(tok, ".")
	raw, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var payload map[string]any
	_ = json.Unmarshal(raw, &payload)
	payload["pca"] = float64(99999)
	forged, _ := json.Marshal(payload)
	parts[1] = base64.RawURLEncoding.EncodeToString(forged)
	if _, _, ok := VerifyToken(secret, strings.Join(parts, ".")); ok {
		t.Fatal("forged pca token should fail signature check")
	}
	// 错误密钥拒绝
	if _, _, ok := VerifyToken("wrong-secret", tok); ok {
		t.Fatal("wrong secret should fail")
	}
	// 过期 token 拒绝
	expired := MakeToken(secret, "admin", -10, 0)
	if _, _, ok := VerifyToken(secret, expired); ok {
		t.Fatal("expired token should fail")
	}
}

// TestTokenPCACompat 旧格式 token(无 pca 字段,签发于引入 SEC-003 之前)按 pca=0 处理:
// 与用户 pca=0(从未变更凭据)匹配,升级后已有会话不失效。
func TestTokenPCACompat(t *testing.T) {
	secret := "test-secret-123"
	now := time.Now().Unix()
	header := b64([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(map[string]any{"sub": "admin", "iat": now, "exp": now + 3600})
	payload := b64(payloadBytes)
	signing := header + "." + payload
	legacy := signing + "." + sign(secret, signing)

	user, pca, ok := VerifyToken(secret, legacy)
	if !ok || user != "admin" || pca != 0 {
		t.Fatalf("legacy token = (%q, %d, %v), want (admin, 0, true)", user, pca, ok)
	}
}
