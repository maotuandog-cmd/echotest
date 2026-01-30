package utils

import (
	"context"
	"echotest/config"
	"echotest/pkg/ratelimit"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func Init(cfg *config.Config) (*echo.Echo, context.Context, context.CancelFunc) {
	ec := echo.New()
	InitLogger()
	ec.Logger = SlogLogger
	// 最先挂载：为每个请求生成或透传 X-Request-Id，便于按 ID 查整条链路日志
	ec.Use(middleware.RequestID())
	ec.Use(middleware.Recover())
	ec.Use(RequestLoggerWithZap())
	ec.Validator = NewCustomValidator()
	ec.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete},
	}))
	bodyLimit := int64(cfg.Server.BodyLimit)
	rateLimitRate, rateLimitBurst := cfg.Server.RateLimitRate, cfg.Server.RateLimitBurst
	if cfg != nil && cfg.Server != nil {
		if cfg.Server.BodyLimit > 0 {
			bodyLimit = cfg.Server.BodyLimit
		}
		if cfg.Server.RateLimitRate > 0 {
			rateLimitRate = cfg.Server.RateLimitRate
		}
		if cfg.Server.RateLimitBurst > 0 {
			rateLimitBurst = cfg.Server.RateLimitBurst
		}
	}
	ec.Use(middleware.BodyLimit(bodyLimit))
	ec.Use(ratelimit.New(ratelimit.Config{Rate: rateLimitRate, Burst: rateLimitBurst}).Middleware())
	ec.Use(middleware.Gzip())
	ec.Use(middleware.Secure())
	// 注意：CSRF 在没有配置的情况下在 v5 中可能也需要具体配置，这里保持默认
	ec.Use(middleware.CSRF())
	ec.Use(echoprometheus.NewMiddleware("echotest"))
	// 指标监控服务
	go func() {
		metrics := echo.New()
		//metrics.HideBanner = true // 这里的子服务通常不需要 banner
		metrics.GET("/metrics", echoprometheus.NewHandler())
		if err := metrics.Start(":8081"); err != nil && err != http.ErrServerClosed {
			ec.Logger.Error("failed to start metrics server", "error", err)
		}
	}()
	// 关键修改：不要在这里 defer cancel()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	return ec, ctx, cancel
}
