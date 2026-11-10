package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnknownProfileFails(t *testing.T) {
	base := t.TempDir()
	t.Setenv("APP_BASE_DIR", base)
	t.Setenv("APP_CONFIG_DIR", base)
	t.Setenv("APP_DATA_DIR", filepath.Join(base, "data"))
	t.Setenv("APP_PROFILE", "staging")
	t.Setenv("SSL_CONFIG", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "profile") {
		t.Fatalf("未知 profile 应失败: %v", err)
	}
}

func TestMissingProfileFileFails(t *testing.T) {
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "application.yaml"), []byte("app:\n  name: demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("APP_BASE_DIR", base)
	t.Setenv("APP_CONFIG_DIR", base)
	t.Setenv("APP_DATA_DIR", filepath.Join(base, "data"))
	t.Setenv("APP_PROFILE", "pro")
	t.Setenv("SSL_CONFIG", "")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "application-pro.yaml") {
		t.Fatalf("缺少环境文件应失败: %v", err)
	}
}

func TestSQLitePathAndLegacyAddress(t *testing.T) {
	base := t.TempDir()
	data := filepath.Join(base, "data")
	t.Setenv("APP_BASE_DIR", base)
	t.Setenv("APP_CONFIG_DIR", base)
	t.Setenv("APP_DATA_DIR", data)
	t.Setenv("APP_PROFILE", "")
	t.Setenv("SSL_CONFIG", "")
	t.Setenv("SSL_DATABASE_PATH", "")
	t.Setenv("SSL_SERVER_ADDR", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Path != filepath.Join(data, "sslplatform.db") {
		t.Fatalf("path=%s", cfg.Database.Path)
	}
	if cfg.Database.SQLite.JournalMode != "WAL" {
		t.Fatalf("journal=%s", cfg.Database.SQLite.JournalMode)
	}
	if cfg.Server.Addr != "127.0.0.1:8080" {
		t.Fatalf("addr=%s", cfg.Server.Addr)
	}

	custom := filepath.Join(base, "custom.db")
	t.Setenv("SSL_DATABASE_PATH", custom)
	t.Setenv("SSL_SERVER_ADDR", ":8080")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Path != custom || cfg.Server.Addr != ":8080" {
		t.Fatalf("path=%s addr=%s", cfg.Database.Path, cfg.Server.Addr)
	}
}
