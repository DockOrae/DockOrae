package service

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"testing"
	"time"
)

// testInjectKey 生成测试密钥对并临时注入公钥注册表,返回 (公钥PEM, 私钥)。
func testInjectKey(t *testing.T, keyID string) (string, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))
	licensePublicKeys[keyID] = pubPEM
	t.Cleanup(func() { delete(licensePublicKeys, keyID) })
	return pubPEM, priv
}

// testSignV2 用私钥签发 V2 Key。
func testSignV2(t *testing.T, priv ed25519.PrivateKey, payload map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	sig := ed25519.Sign(priv, raw)
	return base64.RawURLEncoding.EncodeToString(raw) + "." +
		base64.RawURLEncoding.EncodeToString(sig)
}

func v2Payload(keyID string) map[string]any {
	return map[string]any{
		"version":     2,
		"key_id":      keyID,
		"license_id":  "DMG-TEST01",
		"product":     "docker-manager-go",
		"plan":        "pro",
		"features":    []string{"compose", "container_create"},
		"customer":    "Test",
		"issued_at":   time.Now().Unix(),
		"expires_at":  time.Now().Unix() + 86400,
		"max_devices": 3,
	}
}

// TestLicenseVerifyV2 签发端(Ed25519 私钥)签发 → 消费端(公钥)离线验证通过。
func TestLicenseVerifyV2(t *testing.T) {
	_, priv := testInjectKey(t, "test-01")
	key := testSignV2(t, priv, v2Payload("test-01"))

	info, ok := LicenseVerifyKey(key)
	if !ok {
		t.Fatal("V2 key must verify")
	}
	if strOr(info["status"]) != "active" {
		t.Fatalf("status = %v, want active", info["status"])
	}
	if strOr(info["license_id"]) != "DMG-TEST01" {
		t.Fatalf("license_id = %v", info["license_id"])
	}
	if strOr(info["plan"]) != "pro" {
		t.Fatalf("plan = %v", info["plan"])
	}
}

// TestLicenseVerifyV2TamperedPayload 篡改 payload → 拒绝(签名不匹配)。
func TestLicenseVerifyV2TamperedPayload(t *testing.T) {
	_, priv := testInjectKey(t, "test-02")
	key := testSignV2(t, priv, v2Payload("test-02"))

	parts := splitKey(key)
	raw, _ := base64.RawURLEncoding.DecodeString(parts[0])
	raw = append(raw[:len(raw)-1], 'X') // 篡改最后一个字节
	parts[0] = base64.RawURLEncoding.EncodeToString(raw)
	if _, ok := LicenseVerifyKey(parts[0] + "." + parts[1]); ok {
		t.Fatal("tampered payload must be rejected")
	}
}

// TestLicenseVerifyV2TamperedSignature 篡改签名 → 拒绝。
func TestLicenseVerifyV2TamperedSignature(t *testing.T) {
	_, priv := testInjectKey(t, "test-03")
	key := testSignV2(t, priv, v2Payload("test-03"))

	parts := splitKey(key)
	sigRaw, _ := base64.RawURLEncoding.DecodeString(parts[1])
	sigRaw[0] ^= 0xFF
	parts[1] = base64.RawURLEncoding.EncodeToString(sigRaw)
	if _, ok := LicenseVerifyKey(parts[0] + "." + parts[1]); ok {
		t.Fatal("tampered signature must be rejected")
	}
}

// TestLicenseVerifyV2WrongKey 用错误公钥签发的 Key → 拒绝。
func TestLicenseVerifyV2WrongKey(t *testing.T) {
	// 注入 A,用 B 签发
	testInjectKey(t, "test-a")
	_, privB := testInjectKey(t, "test-b")
	key := testSignV2(t, privB, v2Payload("test-a"))

	if _, ok := LicenseVerifyKey(key); ok {
		t.Fatal("key signed by wrong key must be rejected")
	}
}

// TestLicenseVerifyV2UnknownKeyID 未知 key_id → 拒绝(不静默接受)。
func TestLicenseVerifyV2UnknownKeyID(t *testing.T) {
	_, priv := testInjectKey(t, "test-04")
	key := testSignV2(t, priv, v2Payload("unknown-key-id"))
	if _, ok := LicenseVerifyKey(key); ok {
		t.Fatal("unknown key_id must be rejected")
	}
}

// TestLicenseVerifyV2UnsupportedVersion version != 2 → 拒绝。
func TestLicenseVerifyV2UnsupportedVersion(t *testing.T) {
	_, priv := testInjectKey(t, "test-05")
	p := v2Payload("test-05")
	p["version"] = 99
	key := testSignV2(t, priv, p)
	if _, ok := LicenseVerifyKey(key); ok {
		t.Fatal("unsupported version must be rejected")
	}
}

// TestLicenseVerifyV2UnknownFeature 未知 feature → 拒绝。
func TestLicenseVerifyV2UnknownFeature(t *testing.T) {
	_, priv := testInjectKey(t, "test-06")
	p := v2Payload("test-06")
	p["features"] = []string{"compose", "hack_feature"}
	key := testSignV2(t, priv, p)
	if _, ok := LicenseVerifyKey(key); ok {
		t.Fatal("unknown feature must be rejected")
	}
}

// TestLicenseVerifyV2WrongProduct product 不匹配 → 拒绝。
func TestLicenseVerifyV2WrongProduct(t *testing.T) {
	_, priv := testInjectKey(t, "test-07")
	p := v2Payload("test-07")
	p["product"] = "other-app"
	key := testSignV2(t, priv, p)
	if _, ok := LicenseVerifyKey(key); ok {
		t.Fatal("wrong product must be rejected")
	}
}

// TestLicenseVerifyV2InvalidMaxDevices max_devices < 1 → 拒绝。
func TestLicenseVerifyV2InvalidMaxDevices(t *testing.T) {
	_, priv := testInjectKey(t, "test-08")
	p := v2Payload("test-08")
	p["max_devices"] = 0
	key := testSignV2(t, priv, p)
	if _, ok := LicenseVerifyKey(key); ok {
		t.Fatal("max_devices < 1 must be rejected")
	}
}

// TestLicenseVerifyV2EmptyFeatures 空 features = 无商业功能(不再视为全功能)。
func TestLicenseVerifyV2EmptyFeatures(t *testing.T) {
	_, priv := testInjectKey(t, "test-09")
	p := v2Payload("test-09")
	p["features"] = []string{}
	key := testSignV2(t, priv, p)
	info, ok := LicenseVerifyKey(key)
	if !ok {
		t.Fatal("valid key with empty features must parse")
	}
	if strOr(info["status"]) != "active" {
		t.Fatalf("status = %v", info["status"])
	}
}

// splitKey 拆分 key 为 [payload, sig]。
func splitKey(key string) []string {
	for i := 0; i < len(key); i++ {
		if key[i] == '.' {
			return []string{key[:i], key[i+1:]}
		}
	}
	return []string{key, ""}
}
