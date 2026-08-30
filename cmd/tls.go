package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"os"
	"strings"
)

// tlsErrFilter 丢弃 TLS 握手错误日志:公网 443 会被全网扫描器(Shodan/Censys/僵尸网络)
// 持续探测,Go http.Server 默认把每次失败的握手写入 ErrorLog,这些噪音无运维价值。
// 非握手错误(如 accept 失败)仍写入 stderr,保证 docker logs 可见真实故障。
type tlsErrFilter struct{}

func (tlsErrFilter) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("TLS handshake error")) {
		return len(p), nil
	}
	return os.Stderr.Write(p)
}

// makeGetCertificate 返回基于 SNI 白名单的证书选择器:
//   - webDomain 为空:不启用白名单,接受任意 SNI(与旧行为一致);
//   - webDomain 已设置:仅 SNI == 域名 或 localhost/127.0.0.1/::1(回环运维)放行,
//     其余(IP 直连 SNI 为空、陌生域名、扫描器探测)直接拒绝握手。
//
// 每次握手实时加载证书文件,acme.sh 续期后无需重启面板即可生效。
func makeGetCertificate(domain, certFile, keyFile string) func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	restrict := strings.TrimSpace(domain) != ""
	allowed := map[string]bool{
		"localhost": true,
		"127.0.0.1": true,
		"::1":       true,
	}
	if d := strings.ToLower(strings.TrimSpace(domain)); d != "" {
		allowed[d] = true
	}
	return func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
		sni := strings.ToLower(strings.TrimSpace(hello.ServerName))
		if restrict && !allowed[sni] {
			return nil, fmt.Errorf("rejected: SNI %q not allowed", hello.ServerName)
		}
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, err
		}
		return &cert, nil
	}
}
