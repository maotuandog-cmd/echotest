// Package ratelimit 提供基于令牌桶的接口限流工具，支持按 IP/用户等维度限流，并提供 Echo 中间件。
package ratelimit

import (
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v5"
	"golang.org/x/time/rate"
)

// Config 限流配置
type Config struct {
	// Rate 每秒允许的请求数（令牌生成速率）
	Rate float64
	// Burst 突发允许的最大请求数（桶容量）
	Burst int
	// KeyFunc 从请求中提取限流 key，默认按 RealIP
	KeyFunc func(*echo.Context) string
}

// Limiter 按 key 维度的限流器（如按 IP）
type Limiter struct {
	cfg      Config
	limiters sync.Map // key string -> *rate.Limiter
}

// New 根据配置创建限流器
func New(cfg Config) *Limiter {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c *echo.Context) string {
			return c.RealIP()
		}
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 1
	}
	return &Limiter{cfg: cfg}
}

// Allow 判断该 key 是否在限流内，true 表示允许
func (l *Limiter) Allow(key string) bool {
	lim, _ := l.limiters.LoadOrStore(key, rate.NewLimiter(rate.Limit(l.cfg.Rate), l.cfg.Burst))
	return lim.(*rate.Limiter).Allow()
}

// Reserve 预留一个令牌，返回需等待的时长；d > 0 表示需等待
func (l *Limiter) Reserve(key string) (wait time.Duration, ok bool) {
	lim, _ := l.limiters.LoadOrStore(key, rate.NewLimiter(rate.Limit(l.cfg.Rate), l.cfg.Burst))
	r := lim.(*rate.Limiter).Reserve()
	if !r.OK() {
		return 0, false
	}
	return r.Delay(), true
}

// Middleware 返回 Echo 限流中间件：超限时返回 429，并设置 Retry-After
func (l *Limiter) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			key := l.cfg.KeyFunc(c)
			if !l.Allow(key) {
				c.Response().Header().Set("Retry-After", "1")
				return c.JSON(http.StatusTooManyRequests, map[string]string{
					"message": "rate limit exceeded",
				})
			}
			return next(c)
		}
	}
}
