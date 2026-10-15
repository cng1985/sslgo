package etl

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sslplatform/internal/datasource"
)

func TestDeployRestoresCertificateWhenNginxTestFails(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "certs")
	if err := os.Mkdir(certDir, 0o750); err != nil {
		t.Fatal(err)
	}
	bin := writeNginx(t, dir, true)
	crt := filepath.Join(certDir, "example.com.pem")
	if err := os.WriteFile(crt, []byte("OLD"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := NewDeployer().Deploy(context.Background(), DeployRequest{
		Nginx: datasource.NginxConfig{
			NginxBin: bin, ServiceManager: datasource.ManagerSignal, Action: datasource.ActionRestart,
			Webroot: dir, CertDir: certDir, TestBefore: true,
		},
		CertificateID: 7,
		Domains:       []string{"example.com"},
		CertPEM:       "NEWCERT",
		KeyPEM:        "NEWKEY",
		IssuerPEM:     "NEWCHAIN",
	})
	if err == nil {
		t.Fatal("nginx -t 失败时应当中止部署")
	}
	got, err := os.ReadFile(crt)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "OLD" {
		t.Fatalf("原证书应被恢复，实际 %q", got)
	}
}

func TestDeployWritesCertificateAndRestartsNginx(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "certs")
	snippetDir := filepath.Join(dir, "conf.d")
	if err := os.MkdirAll(certDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(snippetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	kept := filepath.Join(snippetDir, "nbsaas.com.conf")
	if err := os.WriteFile(kept, []byte("server { ssl_certificate nbsaas.com.pem; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := writeNginx(t, dir, false)
	result, err := NewDeployer().Deploy(context.Background(), DeployRequest{
		Nginx: datasource.NginxConfig{
			NginxBin: bin, ServiceManager: datasource.ManagerSignal, Action: datasource.ActionRestart,
			Webroot: dir, CertDir: certDir, SnippetDir: snippetDir, TestBefore: true,
		},
		CertificateID: 3,
		Domains:       []string{"www.example.com"},
		CertPEM:       "CERTDATA",
		KeyPEM:        "KEYDATA",
		IssuerPEM:     "CHAINDATA",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != datasource.ActionRestart {
		t.Fatalf("action=%s", result.Action)
	}
	if result.SnippetPath != "" || result.ChainPath != "" {
		t.Fatalf("不应返回配置或证书链路径: %+v", result)
	}
	got, err := os.ReadFile(result.CertPath)
	if err != nil || string(got) != "CERTDATA\nCHAINDATA\n" {
		t.Fatalf("cert=%q err=%v", got, err)
	}
	if result.CertPath != filepath.Join(certDir, "www.example.com.pem") || result.KeyPath != filepath.Join(certDir, "www.example.com.key") {
		t.Fatalf("paths=%s %s", result.CertPath, result.KeyPath)
	}
	info, err := os.Stat(result.KeyPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("私钥权限 = %o", info.Mode().Perm())
	}
	conf, err := os.ReadFile(kept)
	if err != nil || string(conf) != "server { ssl_certificate nbsaas.com.pem; }\n" {
		t.Fatalf("nginx 配置被修改: %q err=%v", conf, err)
	}
	entries, err := os.ReadDir(snippetDir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("配置目录应保持原样: %v %v", entries, err)
	}
	log, err := os.ReadFile(filepath.Join(dir, "nginx.log"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(log)
	if !strings.Contains(text, "-t") || !strings.Contains(text, "-s quit") {
		t.Fatalf("应先检测再重启，日志 %s", text)
	}
}

func TestDeployOverwritesDomainCertificate(t *testing.T) {
	dir := t.TempDir()
	certDir := filepath.Join(dir, "certs")
	if err := os.MkdirAll(certDir, 0o750); err != nil {
		t.Fatal(err)
	}
	pemPath := filepath.Join(certDir, "nbsaas.com.pem")
	keyPath := filepath.Join(certDir, "nbsaas.com.key")
	if err := os.WriteFile(pemPath, []byte("OLD CERT\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keyPath, []byte("OLD KEY\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	bin := writeNginx(t, dir, false)
	result, err := NewDeployer().Deploy(context.Background(), DeployRequest{
		Nginx: datasource.NginxConfig{
			NginxBin: bin, ServiceManager: datasource.ManagerSignal, Action: datasource.ActionRestart,
			Webroot: dir, CertDir: certDir, TestBefore: true,
		},
		Domains:   []string{"*.nbsaas.com", "nbsaas.com"},
		CertPEM:   "NEW CERT",
		KeyPEM:    "NEW KEY",
		IssuerPEM: "NEW CHAIN",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.CertPath != pemPath || result.KeyPath != keyPath {
		t.Fatalf("paths=%s %s", result.CertPath, result.KeyPath)
	}
	pem, err := os.ReadFile(pemPath)
	if err != nil || string(pem) != "NEW CERT\nNEW CHAIN\n" {
		t.Fatalf("pem=%q err=%v", pem, err)
	}
	key, err := os.ReadFile(keyPath)
	if err != nil || string(key) != "NEW KEY\n" {
		t.Fatalf("key=%q err=%v", key, err)
	}
	entries, err := os.ReadDir(certDir)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, entry := range entries {
		names[entry.Name()] = true
	}
	if !names["nbsaas.com.pem"] || !names["nbsaas.com.key"] {
		t.Fatalf("应覆盖域名证书: %v", names)
	}
	for name := range names {
		if name == "nbsaas.com.pem" || name == "nbsaas.com.key" || strings.Contains(name, ".bak.") {
			continue
		}
		t.Fatalf("证书目录出现了额外文件 %s", name)
	}
}

func writeNginx(t *testing.T, dir string, failTest bool) string {
	t.Helper()
	path := filepath.Join(dir, "nginx")
	body := "#!/bin/sh\necho \"$*\" >> \"" + filepath.Join(dir, "nginx.log") + "\"\n"
	if failTest {
		body += "if [ \"$1\" = \"-t\" ]; then\n  exit 1\nfi\n"
	}
	body += "exit 0\n"
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}
