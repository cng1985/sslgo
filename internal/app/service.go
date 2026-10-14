package app

import "go.uber.org/fx"

// Run 启动依赖注入容器和 HTTP 服务。
func Run() {
	fx.New(Module).Run()
}
