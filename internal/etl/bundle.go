package etl

import (
	"archive/zip"
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"software.sslmate.com/src/go-pkcs12"

	"sslplatform/internal/common/apperror"
	"sslplatform/internal/common/proc"
)

const (
	FormatNginx = "nginx"
	FormatHTTPD = "httpd"
	FormatIIS   = "iis"
)

type bundleFile struct {
	Name string
	Body []byte
}

type bundleInput struct {
	Format    string
	Name      string
	Domains   []string
	NotAfter  time.Time
	CertPEM   string
	KeyPEM    string
	IssuerPEM string
	Password  string
}

func normalizeBundleFormat(format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case FormatNginx:
		return FormatNginx, nil
	case FormatHTTPD, "apache":
		return FormatHTTPD, nil
	case FormatIIS:
		return FormatIIS, nil
	default:
		return "", apperror.New(400, "下载格式只支持 nginx、httpd、iis")
	}
}

func buildBundle(in bundleInput) (*bundleFile, error) {
	format, err := normalizeBundleFormat(in.Format)
	if err != nil {
		return nil, err
	}
	if format == FormatIIS {
		if err := validBundlePassword(in.Password); err != nil {
			return nil, err
		}
	}
	key, err := parsePrivateKey(in.KeyPEM)
	if err != nil {
		return nil, apperror.New(400, "证书私钥无效，无法下载")
	}
	certs, err := parseCertificates(in.CertPEM, in.IssuerPEM)
	if err != nil {
		return nil, apperror.New(400, "证书内容无效，无法下载")
	}
	leaf, chain, err := splitLeaf(certs, key)
	if err != nil {
		return nil, apperror.New(400, "证书与私钥不匹配")
	}
	base := bundleBase(in.Name, in.Domains)
	expiry := formatExpiry(in.NotAfter)
	names := strings.Join(in.Domains, " ")
	var files []zipEntry
	switch format {
	case FormatNginx:
		files = []zipEntry{
			{name: "fullchain.pem", body: []byte(encodeCerts(append([]*x509.Certificate{leaf}, chain...)))},
			{name: "privkey.pem", body: []byte(normalizePEM(in.KeyPEM))},
			{name: "nginx.conf", body: []byte(nginxBundleConf(names, expiry))},
		}
	case FormatHTTPD:
		files = []zipEntry{
			{name: "server.crt", body: []byte(encodeCerts([]*x509.Certificate{leaf}))},
			{name: "server.key", body: []byte(normalizePEM(in.KeyPEM))},
		}
		if len(chain) > 0 {
			files = append(files, zipEntry{name: "chain.crt", body: []byte(encodeCerts(chain))})
		}
		files = append(files, zipEntry{name: "httpd.conf", body: []byte(httpdBundleConf(names, expiry, len(chain) > 0))})
	default:
		pfx, err := pkcs12.LegacyDES.Encode(key, leaf, chain, in.Password)
		if err != nil {
			return nil, apperror.New(400, "生成 IIS 证书失败")
		}
		files = []zipEntry{
			{name: "certificate.pfx", body: pfx},
			{name: "iis.txt", body: []byte(iisBundleNote(names, expiry))},
		}
	}
	body, err := writeZip(files)
	if err != nil {
		return nil, err
	}
	return &bundleFile{Name: base + "-" + format + ".zip", Body: body}, nil
}

func validBundlePassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < 8 || n > 64 {
		return apperror.New(400, "IIS 证书密码需要 8 到 64 位")
	}
	if strings.ContainsAny(password, "\x00\n\r") {
		return apperror.New(400, "IIS 证书密码包含无效字符")
	}
	return nil
}

func readMaterialFiles(certDir string, id uint) (string, string, string, error) {
	dir, err := proc.SafeDir(certDir)
	if err != nil {
		return "", "", "", err
	}
	crt, key, chain := artifactPaths(dir, id)
	certBytes, err := os.ReadFile(crt)
	if err != nil {
		return "", "", "", err
	}
	keyBytes, err := os.ReadFile(key)
	if err != nil {
		return "", "", "", err
	}
	chainBytes, _ := os.ReadFile(chain)
	return string(certBytes), string(keyBytes), string(chainBytes), nil
}

type zipEntry struct {
	name string
	body []byte
}

func writeZip(files []zipEntry) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, file := range files {
		w, err := zw.Create(file.name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(file.body); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func parsePrivateKey(pemText string) (crypto.PrivateKey, error) {
	rest := []byte(pemText)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		switch block.Type {
		case "PRIVATE KEY":
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, err
			}
			priv, ok := key.(crypto.PrivateKey)
			if !ok {
				return nil, fmt.Errorf("不支持的私钥")
			}
			return priv, nil
		case "RSA PRIVATE KEY":
			return x509.ParsePKCS1PrivateKey(block.Bytes)
		case "EC PRIVATE KEY":
			return x509.ParseECPrivateKey(block.Bytes)
		}
	}
	return nil, fmt.Errorf("私钥 PEM 无效")
}

func parseCertificates(chunks ...string) ([]*x509.Certificate, error) {
	var out []*x509.Certificate
	for _, chunk := range chunks {
		rest := []byte(chunk)
		for {
			var block *pem.Block
			block, rest = pem.Decode(rest)
			if block == nil {
				break
			}
			if block.Type != "CERTIFICATE" || len(block.Bytes) == 0 {
				continue
			}
			certs, err := x509.ParseCertificates(block.Bytes)
			if err != nil {
				return nil, err
			}
			out = append(out, certs...)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("证书 PEM 无效")
	}
	return out, nil
}

func splitLeaf(certs []*x509.Certificate, key crypto.PrivateKey) (*x509.Certificate, []*x509.Certificate, error) {
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, nil, fmt.Errorf("不支持的私钥")
	}
	var leaf *x509.Certificate
	for _, cert := range certs {
		if samePublicKey(cert.PublicKey, signer.Public()) {
			leaf = cert
			break
		}
	}
	if leaf == nil {
		return nil, nil, fmt.Errorf("找不到与私钥匹配的证书")
	}
	seen := map[string]struct{}{string(leaf.Raw): {}}
	chain := make([]*x509.Certificate, 0, len(certs))
	for _, cert := range certs {
		if _, ok := seen[string(cert.Raw)]; ok {
			continue
		}
		seen[string(cert.Raw)] = struct{}{}
		chain = append(chain, cert)
	}
	return leaf, chain, nil
}

func samePublicKey(a, b crypto.PublicKey) bool {
	switch pub := a.(type) {
	case *ecdsa.PublicKey:
		other, ok := b.(*ecdsa.PublicKey)
		return ok && pub.Equal(other)
	case *rsa.PublicKey:
		other, ok := b.(*rsa.PublicKey)
		return ok && pub.Equal(other)
	case ed25519.PublicKey:
		other, ok := b.(ed25519.PublicKey)
		return ok && pub.Equal(other)
	default:
		return false
	}
}

func encodeCerts(certs []*x509.Certificate) string {
	var b strings.Builder
	for _, cert := range certs {
		b.Write(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}))
	}
	return b.String()
}

func normalizePEM(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	return text + "\n"
}

func bundleBase(name string, domains []string) string {
	base := strings.TrimSpace(name)
	if base == "" && len(domains) > 0 {
		base = domains[0]
	}
	var b strings.Builder
	for _, r := range base {
		switch {
		case r < 32 || r == 127:
			continue
		case strings.ContainsRune(`/\:*?"<>|`, r) || r == ' ':
			b.WriteRune('-')
		default:
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "-.")
	if out == "" {
		return "certificate"
	}
	runes := []rune(out)
	if len(runes) > 80 {
		out = string(runes[:80])
	}
	return out
}

func formatExpiry(t time.Time) string {
	if t.IsZero() {
		return "未知"
	}
	return t.UTC().Format("2006-01-02 15:04:05 UTC")
}

func nginxBundleConf(domains, expiry string) string {
	return fmt.Sprintf(`# Nginx
# 域名：%s
# 到期时间：%s
ssl_certificate     fullchain.pem;
ssl_certificate_key privkey.pem;
`, domains, expiry)
}

func httpdBundleConf(domains, expiry string, hasChain bool) string {
	chain := "# 此证书没有中间证书，不需要 SSLCertificateChainFile。\n"
	if hasChain {
		chain = "SSLCertificateChainFile chain.crt\n"
	}
	return fmt.Sprintf(`# Apache httpd
# 域名：%s
# 到期时间：%s
SSLCertificateFile      server.crt
SSLCertificateKeyFile   server.key
%s`, domains, expiry, chain)
}

func iisBundleNote(domains, expiry string) string {
	return fmt.Sprintf(`IIS 导入说明
域名：%s
到期时间：%s
1. 打开 IIS 管理器，选中服务器节点，打开“服务器证书”。
2. 点击“导入”，选择同目录下的 certificate.pfx。
3. 输入下载时填写的密码。PFX 使用 IIS 可导入的传统 PKCS#12。
4. 在网站绑定中新增 https，并选择这张证书。
`, domains, expiry)
}

func remainingDays(notAfter, now time.Time) int {
	d := notAfter.Sub(now)
	if d >= 0 {
		return int(d / (24 * time.Hour))
	}
	passed := -d
	days := int(passed / (24 * time.Hour))
	if passed%(24*time.Hour) != 0 {
		days++
	}
	if days == 0 {
		days = 1
	}
	return -days
}
