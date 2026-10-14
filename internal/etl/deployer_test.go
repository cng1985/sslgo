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
	crt := filepath.Join(certDir, "cert-7.crt")
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
	got, err := os.ReadFile(result.CertPath)
	if err != nil || string(got) != "CERTDATA" {
		t.Fatalf("cert=%q err=%v", got, err)
	}
	info, err := os.Stat(result.KeyPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("私钥权限 = %o", info.Mode().Perm())
	}
	snippet, err := os.ReadFile(result.SnippetPath)
	if err != nil || !strings.Contains(string(snippet), "www.example.com") {
		t.Fatalf("snippet=%q err=%v", snippet, err)
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
