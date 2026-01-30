// router.go 定义了应用程序的所有HTTP路由
package app

import (
	"net/http"

	"echotest/pkg/requestid"

	"github.com/labstack/echo/v5"
)

// initRouter 初始化所有HTTP路由
func (a *Application) initRouter() {
	// 测试日志与请求 ID：响应中返回 request_id，便于用该 ID 在日志中检索整条链路
	a.E.GET("/ping", func(c *echo.Context) error {
		rid := requestid.GetRequestID(c)
		a.E.Logger.Info("ping called", "request_id", rid)
		return c.JSON(http.StatusOK, map[string]string{
			"status":     "ok",
			"request_id": rid,
		})
	})
}
