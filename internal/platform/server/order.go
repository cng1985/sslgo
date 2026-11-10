package server

// BeforeHTTP 只用于启动顺序：实现者先注册自己的 OnStart，HTTP 再开始监听。
type BeforeHTTP interface {
	MarkBeforeHTTP()
}

// StartupOrder 是 BeforeHTTP 的空实现。
type StartupOrder struct{}

func (StartupOrder) MarkBeforeHTTP() {}
