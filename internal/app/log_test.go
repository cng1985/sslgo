package app_test

import (
	"bytes"
	"context"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"sslplatform/internal/app"
	"sslplatform/internal/logging"
	"sslplatform/internal/server"
)

func TestStartupLogFormat(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(logging.NewHandler(&buf)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	t.Setenv("SSL_DATABASE_PATH", filepath.Join(dir, "app.db"))
	t.Setenv("SSL_SERVER_ADDR", "127.0.0.1:0")
	t.Setenv("SSL_SERVER_MODE", "test")

	var endpoint *server.Endpoint
	application := fx.New(
		app.Module,
		fx.Populate(&endpoint),
		fx.WithLogger(func() fxevent.Logger { return logging.FxLogger{} }),
	)
	startCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := application.Start(startCtx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer stopCancel()
		_ = application.Stop(stopCtx)
	})
	if endpoint.Get() == "" {
		t.Fatal("服务未监听")
	}

	out := buf.String()
	want := []string{
		"level=INFO msg=数据库迁移完成",
		"level=INFO msg=已初始化默认租户和管理员",
		"level=INFO msg=证枢启动",
		"version=1.0.0",
		"addr=127.0.0.1:0",
		"database=" + filepath.Join(dir, "app.db"),
		`level=WARN msg="正在使用默认管理员密码 Admin@123456，请尽快在系统配置中修改，或通过环境变量 SSL_ADMIN_PASSWORD 设置"`,
		`level=WARN msg="正在使用默认 JWT 密钥，请尽快修改，或通过环境变量 SSL_JWT_SECRET 设置"`,
		"level=INFO msg=调度器已启动",
		`level=INFO msg="HTTP 服务已启动"`,
	}
	last := -1
	for _, part := range want {
		idx := strings.Index(out, part)
		if idx < 0 || idx < last {
			t.Fatalf("缺少或顺序不对 %q\n%s", part, out)
		}
		last = idx
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if !strings.Contains(line, " level=INFO ") && !strings.Contains(line, " level=WARN ") {
			t.Fatalf("日志行格式不对: %s", line)
		}
		if !strings.HasPrefix(line, "time=") {
			t.Fatalf("日志行格式不对: %s", line)
		}
	}
	if strings.Contains(out, "[Fx]") || strings.Contains(out, "ssl platform listening") {
		t.Fatalf("仍有旧日志格式\n%s", out)
	}
}
