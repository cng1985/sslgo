package logging

import (
	"context"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"strings"

	"go.uber.org/fx/fxevent"

	"sslplatform/internal/platform/config"
)

// Setup 把标准库日志和 slog 都写成同一行文本：time= level= msg= key=value。
func Setup(cfg *config.Config) {
	level := slog.LevelInfo
	if cfg != nil {
		switch cfg.Logging.Level {
		case "debug":
			level = slog.LevelDebug
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}
	format := "text"
	if cfg != nil && cfg.Logging.Format != "" {
		format = cfg.Logging.Format
	}
	slog.SetDefault(slog.New(NewHandlerLevel(os.Stdout, level, format)))
	log.SetFlags(0)
	log.SetOutput(slogWriter{level: slog.LevelInfo})
}

// NewHandler 返回和启动日志相同的文本格式。
func NewHandler(w io.Writer) slog.Handler {
	return NewHandlerLevel(w, slog.LevelInfo, "text")
}

func NewHandlerLevel(w io.Writer, level slog.Level, format string) slog.Handler {
	opts := &slog.HandlerOptions{Level: level}
	if format == "json" {
		return slog.NewJSONHandler(w, opts)
	}
	return slog.NewTextHandler(w, opts)
}

// FormatAddr 把通配监听地址显示成 0.0.0.0:port。
func FormatAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	switch host {
	case "", "::", "[::]":
		return "0.0.0.0:" + port
	default:
		return addr
	}
}

type slogWriter struct{ level slog.Level }

func (w slogWriter) Write(p []byte) (int, error) {
	msg := strings.TrimSpace(string(p))
	if msg != "" {
		slog.Log(context.Background(), w.level, msg)
	}
	return len(p), nil
}

// FxLogger 只把依赖注入的失败写进同一套日志。
type FxLogger struct{}

func (FxLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuted:
		if e.Err != nil {
			slog.Error("启动失败", "callee", e.FunctionName, "err", e.Err)
		}
	case *fxevent.OnStopExecuted:
		if e.Err != nil {
			slog.Error("停止失败", "callee", e.FunctionName, "err", e.Err)
		}
	case *fxevent.Invoked:
		if e.Err != nil {
			slog.Error("初始化失败", "callee", e.FunctionName, "err", e.Err)
		}
	case *fxevent.Started:
		if e.Err != nil {
			slog.Error("启动失败", "err", e.Err)
		}
	case *fxevent.Stopped:
		if e.Err != nil {
			slog.Error("停止失败", "err", e.Err)
		}
	case *fxevent.RolledBack:
		if e.Err != nil {
			slog.Error("回滚失败", "err", e.Err)
		}
	}
}

// Apply 在配置就绪后按格式调整日志。测试已经设置好的文本日志不会被默认格式替换。
func Apply(cfg *config.Config) {
	if cfg == nil || cfg.Logging.Format != "json" {
		return
	}
	Setup(cfg)
}
