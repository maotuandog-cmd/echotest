// Package tracing 基于 Echo Jaeger 中间件封装链路追踪，便于按请求查看整条调用链。
// 参考: https://echo.labstack.com/docs/middleware/jaeger
//
// 配置通过环境变量（Jaeger 标准）：
//   - JAEGER_SERVICE_NAME    服务名，默认可用程序名
//   - JAEGER_AGENT_HOST       Agent 主机，默认 localhost
//   - JAEGER_AGENT_PORT       Agent 端口，默认 6831
//   - JAEGER_ENDPOINT         HTTP 上报地址，如 http://jaeger:14268/api/traces
//   - JAEGER_DISABLED         true 时关闭追踪
package tracing

import (
	"github.com/labstack/echo-contrib/jaegertracing"
	"github.com/labstack/echo/v5"
)

// Skipper 用于跳过不需要追踪的请求（如 /metrics、/health）
type Skipper func(c *echo.Context) bool

// Register 为 Echo 注册 Jaeger 链路追踪中间件，并为每个请求创建根 span。
// skipper 为 nil 时不跳过任何 URL；返回的 close 应在程序退出时调用（如 defer close()）。
func Register(e *echo.Echo, skipper Skipper) (close func()) {
	c := jaegertracing.New(e, skipper)
	return func() { c.Close() }
}

// TraceFunction 在子 span 中执行 fn，用于追踪某段逻辑的耗时。
// 用法: tracing.TraceFunction(c, myFunc, arg1, arg2)
func TraceFunction(c *echo.Context, fn interface{}, args ...interface{}) {
	jaegertracing.TraceFunction(c, fn, args...)
}

// CreateChildSpan 创建子 span，用于在 Handler 内打点（LogEvent、SetTag、SetBaggageItem 等）。
// 使用完毕后须调用 span.Finish()。
// 用法: sp := tracing.CreateChildSpan(c, "db.query"); defer sp.Finish(); sp.SetTag("query", "SELECT ...")
func CreateChildSpan(c *echo.Context, operationName string) *jaegertracing.Span {
	return jaegertracing.CreateChildSpan(c, operationName)
}
