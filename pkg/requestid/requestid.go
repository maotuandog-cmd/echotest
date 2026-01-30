// Package requestid 基于 Echo 的 X-Request-Id 提供请求 ID 获取与链路日志，便于按请求 ID 查看整条链路的状况。
// 参考: https://echo.labstack.com/docs/middleware/request-id
package requestid

import (
	"github.com/labstack/echo/v5"
	"go.uber.org/zap"
)

// GetRequestID 从当前请求中获取请求 ID（请求头 X-Request-Id 或中间件写入的响应头）。
// 调用方应在 RequestID 中间件之后使用（如 Handler 内）。
func GetRequestID(c *echo.Context) string {
	if c == nil {
		return ""
	}
	id := c.Request().Header.Get(echo.HeaderXRequestID)
	if id != "" {
		return id
	}
	return c.Response().Header().Get(echo.HeaderXRequestID)
}

// LogWithRequestID 使用 zap 记录一条带 request_id 的日志，便于按 ID 在日志中检索整条链路。
func LogWithRequestID(c *echo.Context, logger *zap.Logger, level string, msg string, fields ...zap.Field) {
	id := GetRequestID(c)
	all := make([]zap.Field, 0, len(fields)+1)
	all = append(all, zap.String("request_id", id))
	all = append(all, fields...)
	switch level {
	case "debug":
		logger.Debug(msg, all...)
	case "info":
		logger.Info(msg, all...)
	case "warn":
		logger.Warn(msg, all...)
	case "error":
		logger.Error(msg, all...)
	default:
		logger.Info(msg, all...)
	}
}

// Logger 返回一个在每条日志中自动附带当前请求 request_id 的 logger 封装，用于在链路中打点。
// 用法: requestid.Logger(c, utils.Log).Info("step", zap.String("detail", "xxx"))
func Logger(c *echo.Context, base *zap.Logger) *zap.Logger {
	id := GetRequestID(c)
	return base.With(zap.String("request_id", id))
}
