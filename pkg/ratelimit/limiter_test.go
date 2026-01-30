package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestLimiter_Allow(t *testing.T) {
	// 每秒 10 个，突发 3：同一 key 连续请求，前 3 次允许，第 4 次应拒绝
	l := New(Config{Rate: 10, Burst: 3})
	key := "192.168.1.1"

	for i := 0; i < 3; i++ {
		if !l.Allow(key) {
			t.Errorf("请求 %d: 期望允许，实际被拒绝", i+1)
		}
	}
	if l.Allow(key) {
		t.Error("第 4 次请求: 期望拒绝(超限)，实际允许")
	}
	// 不同 key 不受影响
	if !l.Allow("192.168.1.2") {
		t.Error("不同 key 应允许")
	}
}

func TestLimiter_Middleware_Returns429WhenExceeded(t *testing.T) {
	// 限流：突发 2，同一 IP 第 3 次应返回 429
	lim := New(Config{Rate: 10, Burst: 2})
	e := echo.New()
	e.GET("/test", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, lim.Middleware())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:12345"

	// 前 2 次应 200
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("请求 %d: 期望 200，得到 %d", i+1, rec.Code)
		}
	}
	// 第 3 次应 429
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("第 3 次请求: 期望 429，得到 %d", rec.Code)
	}
	if h := rec.Header().Get("Retry-After"); h != "1" {
		t.Errorf("期望 Retry-After: 1，得到 %q", h)
	}
}

func TestLimiter_Middleware_DifferentIPsIndependent(t *testing.T) {
	lim := New(Config{Rate: 10, Burst: 1})
	e := echo.New()
	e.GET("/test", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}, lim.Middleware())

	// IP1 第 1 次 200
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "10.0.0.1:80"
	rec1 := httptest.NewRecorder()
	e.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Errorf("IP1 第 1 次: 期望 200，得到 %d", rec1.Code)
	}

	// IP2 第 1 次也应 200（按 key 独立计数）
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "10.0.0.2:80"
	rec2 := httptest.NewRecorder()
	e.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("IP2 第 1 次: 期望 200，得到 %d", rec2.Code)
	}

	// IP1 第 2 次应 429（burst=1）
	rec1b := httptest.NewRecorder()
	e.ServeHTTP(rec1b, req1)
	if rec1b.Code != http.StatusTooManyRequests {
		t.Errorf("IP1 第 2 次: 期望 429，得到 %d", rec1b.Code)
	}
}
