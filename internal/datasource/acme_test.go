package datasource

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"sslplatform/internal/acme"
	"sslplatform/internal/config"
)

func TestACMEAccountCreateBindsCloudDNS(t *testing.T) {
	svc := newDatasourceService(t, nil)
	ctx := context.Background()
	view, err := svc.Create(ctx, 1, CreateRequest{
		Name: "阿里云账户",
		Type: TypeACME,
		Config: map[string]any{
			"email":       "ops@example.com",
			"ca":          "letsencrypt",
			"dnsProvider": acme.DNSAliyun,
			"dns": map[string]any{
				"accessKeyId":     "LTAItest",
				"accessKeySecret": "super-secret",
				"region":          "cn-hangzhou",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := mustJSON(t, view.Config)
	if strings.Contains(raw, "super-secret") || strings.Contains(raw, "accountKeyPEM") {
		t.Fatalf("响应泄露密钥: %s", raw)
	}
	if view.Config["accountKeyStored"] != true || view.Config["ca"] != "letsencrypt" {
		t.Fatalf("config=%v", view.Config)
	}
	if view.Config["dnsProvider"] != acme.DNSAliyun {
		t.Fatalf("provider=%v", view.Config["dnsProvider"])
	}
	stored, err := svc.repo.Get(ctx, 1, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := ParseACME(stored.ConfigJSON)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DNS["accessKeySecret"] != "super-secret" || cfg.AccountKeyPEM == "" {
		t.Fatalf("保存的凭证不完整: %+v", cfg.DNS)
	}

	updated, err := svc.Update(ctx, 1, UpdateRequest{
		ID: view.ID,
		Config: map[string]any{
			"email":       "ops@example.com",
			"ca":          "letsencrypt",
			"dnsProvider": acme.DNSAliyun,
			"dns": map[string]any{
				"accessKeyId": "LTAItest",
				"region":      "cn-beijing",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(mustJSON(t, updated.Config), "super-secret") {
		t.Fatal("更新响应不应回显 Secret")
	}
	stored, err = svc.repo.Get(ctx, 1, view.ID)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err = ParseACME(stored.ConfigJSON)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DNS["accessKeySecret"] != "super-secret" || cfg.DNS["region"] != "cn-beijing" {
		t.Fatalf("更新后应保留密钥并改地域: %+v", cfg.DNS)
	}
}

func TestRegisterACMEAccount(t *testing.T) {
	var calls int
	svc := newDatasourceService(t, func(_ context.Context, req acme.RegisterRequest) (*acme.RegisterResult, error) {
		calls++
		if req.Email != "ops@example.com" || req.AccountKeyPEM == "" {
			t.Fatalf("register req=%+v", req)
		}
		return &acme.RegisterResult{AccountKeyPEM: req.AccountKeyPEM, RegistrationURI: "https://acme.example/acme/acct/1"}, nil
	})
	ctx := context.Background()
	view, err := svc.Create(ctx, 1, CreateRequest{
		Name: "待注册",
		Type: TypeACME,
		Config: map[string]any{
			"email": "ops@example.com",
			"ca":    "letsencrypt-staging",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("未要求注册时不应访问 ACME 目录")
	}
	if _, err = svc.Register(ctx, 1, RegisterAccountRequest{ID: view.ID}); err == nil {
		t.Fatal("未同意条款应拒绝")
	}
	registered, err := svc.Register(ctx, 1, RegisterAccountRequest{ID: view.ID, AgreeTOS: true})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || registered.Config["accountRegistered"] != true {
		t.Fatalf("calls=%d config=%v", calls, registered.Config)
	}
	if _, err = svc.Register(ctx, 1, RegisterAccountRequest{ID: view.ID, AgreeTOS: true}); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("已注册账户不应重复请求目录")
	}

	if _, err = svc.Create(ctx, 1, CreateRequest{
		Name: "ZeroSSL",
		Type: TypeACME,
		Config: map[string]any{
			"email": "ops@example.com", "ca": "zerossl", "register": true, "termsAgreed": true,
		},
	}); err == nil {
		t.Fatal("ZeroSSL 缺少 EAB 应拒绝")
	}
}

func TestRejectIncompleteCloudDNS(t *testing.T) {
	svc := newDatasourceService(t, nil)
	_, err := svc.Create(context.Background(), 1, CreateRequest{
		Name: "华为云",
		Type: TypeACME,
		Config: map[string]any{
			"email":       "ops@example.com",
			"dnsProvider": acme.DNSHuawei,
			"dns":         map[string]any{"accessKeyId": "ak", "secretAccessKey": "sk"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "地域") {
		t.Fatalf("err=%v", err)
	}
}

func newDatasourceService(t *testing.T, register func(context.Context, acme.RegisterRequest) (*acme.RegisterResult, error)) *Service {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ds.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&DataSource{}); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	cfg.ACME.DirectoryURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
	cfg.ACME.Timeout = time.Second
	svc := NewService(NewRepository(db), cfg)
	if register != nil {
		svc.register = register
	}
	return svc
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
