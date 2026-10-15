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
)

// Setup 把标准库日志和 slog 都写成同一行文本：time= level= msg= key=value。
func Setup() {
	slog.SetDefault(slog.New(NewHandler(os.Stdout)))
	log.SetFlags(0)
	log.SetOutput(slogWriter{level: slog.LevelInfo})
}

// NewHandler 返回和启动日志相同的文本格式。
func NewHandler(w io.Writer) slog.Handler {
	return slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo})
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

// FxLogger 只把依赖注入的失败写进同一套日志，成功的装配过程不单独占行。
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
