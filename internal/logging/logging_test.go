package logging

import (
	"bytes"
	"log/slog"
	"regexp"
	"testing"
)

func TestTextFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewHandler(&buf))
	logger.Info("数据库迁移完成")
	logger.Info("证枢启动", "version", "1.0.0", "addr", "0.0.0.0:8080", "database", "data/sslplatform.db")
	logger.Warn("正在使用默认管理员密码 Admin@123456，请尽快在系统配置中修改，或通过环境变量 SSL_ADMIN_PASSWORD 设置")

	lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n"))
	if len(lines) != 3 {
		t.Fatalf("lines=%d\n%s", len(lines), buf.String())
	}
	info := regexp.MustCompile(`^time=\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2}) level=INFO msg=数据库迁移完成$`)
	if !info.Match(lines[0]) {
		t.Fatalf("migration line: %s", lines[0])
	}
	start := regexp.MustCompile(`^time=\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2}) level=INFO msg=证枢启动 version=1\.0\.0 addr=0\.0\.0\.0:8080 database=data/sslplatform\.db$`)
	if !start.Match(lines[1]) {
		t.Fatalf("start line: %s", lines[1])
	}
	warn := regexp.MustCompile(`^time=\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?(Z|[+-]\d{2}:\d{2}) level=WARN msg="正在使用默认管理员密码 Admin@123456，请尽快在系统配置中修改，或通过环境变量 SSL_ADMIN_PASSWORD 设置"$`)
	if !warn.Match(lines[2]) {
		t.Fatalf("warn line: %s", lines[2])
	}
}

func TestFormatAddr(t *testing.T) {
	cases := map[string]string{
		":8080":           "0.0.0.0:8080",
		"[::]:8080":       "0.0.0.0:8080",
		"0.0.0.0:8080":    "0.0.0.0:8080",
		"127.0.0.1:8080":  "127.0.0.1:8080",
		"127.0.0.1:51234": "127.0.0.1:51234",
	}
	for in, want := range cases {
		if got := FormatAddr(in); got != want {
			t.Fatalf("FormatAddr(%q)=%q want %q", in, got, want)
		}
	}
}
