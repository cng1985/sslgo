package etl

import (
	"archive/zip"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

func TestBuildBundleFormats(t *testing.T) {
	leafPEM, keyPEM, issuerPEM := mintSignedPair(t)
	expiry := time.Now().Add(40 * 24 * time.Hour).UTC().Truncate(time.Second)
	nginx, err := buildBundle(bundleInput{
		Format: "nginx", Name: "示例 站点", Domains: []string{"example.com", "*.example.com"},
		NotAfter: expiry, CertPEM: leafPEM + issuerPEM, KeyPEM: keyPEM, IssuerPEM: issuerPEM,
	})
	if err != nil {
		t.Fatal(err)
	}
	if nginx.Name != "示例-站点-nginx.zip" {
		t.Fatalf("filename=%s", nginx.Name)
	}
	files := unzipBundle(t, nginx.Body)
	full := certsInPEM(t, string(files["example.com.pem"]))
	if len(full) != 2 {
		t.Fatalf("fullchain certs=%d", len(full))
	}
	if !bytes.Contains(files["nginx.conf"], []byte("ssl_certificate     example.com.pem;")) || !bytes.Contains(files["nginx.conf"], []byte("ssl_certificate_key example.com.key;")) {
		t.Fatalf("nginx.conf=%s", files["nginx.conf"])
	}
	if !bytes.Contains(files["nginx.conf"], []byte(formatExpiry(expiry))) {
		t.Fatalf("缺少到期时间: %s", files["nginx.conf"])
	}
	key, err := parsePrivateKey(string(files["example.com.key"]))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := splitLeaf(full, key); err != nil {
		t.Fatal(err)
	}

	httpd, err := buildBundle(bundleInput{
		Format: "apache", Name: "示例 站点", Domains: []string{"example.com"},
		NotAfter: expiry, CertPEM: leafPEM, KeyPEM: keyPEM, IssuerPEM: issuerPEM,
	})
	if err != nil {
		t.Fatal(err)
	}
	httpdFiles := unzipBundle(t, httpd.Body)
	if len(certsInPEM(t, string(httpdFiles["server.crt"]))) != 1 {
		t.Fatal("httpd 证书文件应只含叶子证书")
	}
	if len(certsInPEM(t, string(httpdFiles["chain.crt"]))) != 1 {
		t.Fatal("httpd 证书链应只含中间证书")
	}
	if !bytes.Contains(httpdFiles["httpd.conf"], []byte("SSLCertificateChainFile chain.crt")) {
		t.Fatalf("httpd.conf=%s", httpdFiles["httpd.conf"])
	}

	iis, err := buildBundle(bundleInput{
		Format: "iis", Name: "示例 站点", Domains: []string{"example.com"},
		NotAfter: expiry, CertPEM: leafPEM, KeyPEM: keyPEM, IssuerPEM: issuerPEM, Password: "Secret@123",
	})
	if err != nil {
		t.Fatal(err)
	}
	iisFiles := unzipBundle(t, iis.Body)
	_, cert, cas, err := pkcs12.DecodeChain(iisFiles["certificate.pfx"], "Secret@123")
	if err != nil {
		t.Fatal(err)
	}
	if len(cas) != 1 {
		t.Fatalf("iis chain=%d", len(cas))
	}
	parsedKey, err := parsePrivateKey(keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := splitLeaf([]*x509.Certificate{cert}, parsedKey); err != nil {
		t.Fatal("PFX 叶子证书与原私钥不匹配")
	}
	if !bytes.Contains(iisFiles["iis.txt"], []byte("服务器证书")) {
		t.Fatalf("iis.txt=%s", iisFiles["iis.txt"])
	}
}

func TestNginxBundleNamesWildcardDomain(t *testing.T) {
	certPEM, keyPEM, err := MintSelfSigned([]string{"*.nbssas.com"})
	if err != nil {
		t.Fatal(err)
	}
	file, err := buildBundle(bundleInput{
		Format: FormatNginx, Name: "通配符", Domains: []string{"*.nbssas.com"},
		CertPEM: certPEM, KeyPEM: keyPEM, IssuerPEM: certPEM,
	})
	if err != nil {
		t.Fatal(err)
	}
	files := unzipBundle(t, file.Body)
	if _, ok := files["nbssas.com.pem"]; !ok {
		t.Fatalf("缺少 nbssas.com.pem: %v", mapsKeys(files))
	}
	if _, ok := files["nbssas.com.key"]; !ok {
		t.Fatalf("缺少 nbssas.com.key: %v", mapsKeys(files))
	}
	if _, ok := files["fullchain.pem"]; ok {
		t.Fatal("不应再使用 fullchain.pem")
	}
	if !bytes.Contains(files["nginx.conf"], []byte("ssl_certificate     nbssas.com.pem;")) || !bytes.Contains(files["nginx.conf"], []byte("ssl_certificate_key nbssas.com.key;")) {
		t.Fatalf("nginx.conf=%s", files["nginx.conf"])
	}
	if !bytes.Contains(files["nbssas.com.key"], []byte("PRIVATE KEY")) {
		t.Fatal("私钥内容无效")
	}
}

func mapsKeys(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	return names
}

func TestSelfSignedBundleOmitsDuplicateChain(t *testing.T) {
	certPEM, keyPEM, err := MintSelfSigned([]string{"solo.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	file, err := buildBundle(bundleInput{
		Format: FormatHTTPD, Name: "solo", Domains: []string{"solo.example.com"},
		CertPEM: certPEM, KeyPEM: keyPEM, IssuerPEM: certPEM,
	})
	if err != nil {
		t.Fatal(err)
	}
	files := unzipBundle(t, file.Body)
	if _, ok := files["chain.crt"]; ok {
		t.Fatal("自签证书不应再附一份重复的证书链")
	}
	if !bytes.Contains(files["httpd.conf"], []byte("不需要 SSLCertificateChainFile")) {
		t.Fatalf("httpd.conf=%s", files["httpd.conf"])
	}
}

func TestBundleRejects(t *testing.T) {
	certPEM, keyPEM, err := MintSelfSigned([]string{"solo.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildBundle(bundleInput{Format: "caddy", CertPEM: certPEM, KeyPEM: keyPEM}); err == nil {
		t.Fatal("应拒绝未知格式")
	}
	if _, err := buildBundle(bundleInput{Format: FormatIIS, CertPEM: certPEM, KeyPEM: keyPEM, Password: "short"}); err == nil {
		t.Fatal("应拒绝过短的 IIS 密码")
	}
	otherPEM, _, err := MintSelfSigned([]string{"other.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildBundle(bundleInput{Format: FormatNginx, CertPEM: otherPEM, KeyPEM: keyPEM}); err == nil {
		t.Fatal("应拒绝不匹配的私钥")
	}
}

func TestReadMaterialFiles(t *testing.T) {
	dir := t.TempDir()
	crt, key, chain := artifactPaths(dir, 7)
	if err := os.WriteFile(crt, []byte("CERT"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(key, []byte("KEY"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chain, []byte("CHAIN"), 0o644); err != nil {
		t.Fatal(err)
	}
	gotCert, gotKey, gotChain, err := readMaterialFiles(dir, 7)
	if err != nil {
		t.Fatal(err)
	}
	if gotCert != "CERT" || gotKey != "KEY" || gotChain != "CHAIN" {
		t.Fatalf("got %s %s %s", gotCert, gotKey, gotChain)
	}
	if _, _, _, err := readMaterialFiles("relative", 7); err == nil {
		t.Fatal("应拒绝相对路径")
	}
	if _, _, _, err := readMaterialFiles(filepath.Join(dir, "missing"), 7); err == nil {
		t.Fatal("缺少文件时应失败")
	}
}

func mintSignedPair(t *testing.T) (string, string, string) {
	t.Helper()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	caTpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTpl, caTpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "example.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(90 * 24 * time.Hour),
		DNSNames:     []string{"example.com", "*.example.com"},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM, err := marshalKeyPEM(leafKey)
	if err != nil {
		t.Fatal(err)
	}
	leafPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER}))
	issuerPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}))
	return leafPEM, keyPEM, issuerPEM
}

func unzipBundle(t *testing.T, body []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, file := range zr.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		out[file.Name] = data
	}
	return out
}

func certsInPEM(t *testing.T, text string) []*x509.Certificate {
	t.Helper()
	certs, err := parseCertificates(text)
	if err != nil {
		t.Fatal(err)
	}
	return certs
}
