package app

import (
	"context"
	"database/sql"
	"echotest/config"
	"echotest/pkg/utils"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"
)

type Application struct {
	Config *config.Config
	E      *echo.Echo
	Ctx    context.Context
	Cancel context.CancelFunc
	Db     *sql.DB
}

// InitApp 加载配置、初始化 Echo（含中间件与日志）、注册路由，返回可运行的 Application
func InitApp(filePath string) (*Application, error) {
	cfg, err := config.NewConfig(filePath)
	if err != nil {
		return nil, err
	}
	ec, ctx, cancel := utils.Init(cfg)
	a := &Application{
		Config: cfg,
		E:      ec,
		Ctx:    ctx,
		Cancel: cancel,
	}
	a.initRouter()
	return a, nil
}

// Run 启动 HTTP 服务并阻塞直到收到退出信号，然后优雅关闭
func (a *Application) Run() error {
	defer a.Cancel()
	s := a.Config.Server
	addr := s.Address + ":" + s.Port
	server := &http.Server{
		Addr:              addr,
		Handler:           a.E,
		ReadTimeout:       orDuration(s.ReadTimeout, 20*time.Second),
		ReadHeaderTimeout: orDuration(s.ReadHeaderTimeout, 5*time.Second),
		WriteTimeout:      orDuration(s.WriteTimeout, 5*time.Second),
		IdleTimeout:       orDuration(s.IdleTimeout, 5*time.Second),
		MaxHeaderBytes:    orInt(s.MaxHeaderBytes, 1<<20),
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			a.E.Logger.Error("failed to start server", "error", err)
		}
	}()
	<-a.Ctx.Done()
	a.E.Logger.Info("shutting down gracefully")
	shutdownTimeout := orDuration(s.ShutdownTimeout, 5*time.Second)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	return server.Shutdown(shutdownCtx)
}

func orDuration(d, def time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return def
}

func orInt(n, def int) int {
	if n > 0 {
		return n
	}
	return def
}
