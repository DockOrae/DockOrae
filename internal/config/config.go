package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	DataDir   string
	Port      uint16
	JWTSecret string
}

func Load() *Config {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "/data"
	}
	port := uint16(8080)
	if p := os.Getenv("PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 && n < 65536 {
			port = uint16(n)
		}
	}
	_ = os.MkdirAll(dataDir, 0o755)

	cfgFile := filepath.Join(dataDir, "config.json")
	jwtSecret := genSecret()
	if raw, err := os.ReadFile(cfgFile); err == nil {
		var v map[string]any
		if json.Unmarshal(raw, &v) == nil {
			if s, ok := v["jwt_secret"].(string); ok && s != "" {
				jwtSecret = s
			}
		}
	}
	out, _ := json.MarshalIndent(map[string]string{"jwt_secret": jwtSecret}, "", "  ")
	_ = os.WriteFile(cfgFile, out, 0o600)

	return &Config{DataDir: dataDir, Port: port, JWTSecret: jwtSecret}
}

func genSecret() string {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	return base64.StdEncoding.EncodeToString(buf)
}
