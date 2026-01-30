package utils

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 创建一个请求日志中间件，兼容 Echo v5
func Logger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		start := time.Now()

		err := next(c)

		req := c.Request()
		res := c.Response()
		latency := time.Since(start)

		// 获取 Echo v5 的 logger（slog.Logger）
		logger := c.Logger()

		// 尝试从 ResponseWriter 中解包 Response 以获取状态码和大小
		var status int = -1
		var size int64 = -1
		if resp, unwrapErr := echo.UnwrapResponse(res); unwrapErr == nil {
			status = resp.Status
			size = resp.Size
		}

		// 如果无法从 Response 获取状态码，尝试从错误中获取
		if status == -1 && err != nil {
			var hsc echo.HTTPStatusCoder
			if errors.As(err, &hsc) {
				status = hsc.StatusCode()
			}
		}

		// 如果仍然没有状态码，默认为 200
		if status == -1 {
			status = 200
		}

		// 构建日志属性
		attrs := []slog.Attr{
			slog.String("remote_ip", c.RealIP()),
			slog.Duration("latency", latency),
			slog.String("host", req.Host),
			slog.String("method", req.Method),
			slog.String("uri", req.RequestURI),
			slog.Int("status", status),
			slog.String("user_agent", req.UserAgent()),
		}

		// 添加响应大小（如果可用）
		if size >= 0 {
			attrs = append(attrs, slog.Int64("size", size))
		}

		// 添加请求 ID（如果存在）
		id := req.Header.Get(echo.HeaderXRequestID)
		if id == "" {
			id = res.Header().Get(echo.HeaderXRequestID)
		}
		if id != "" {
			attrs = append(attrs, slog.String("request_id", id))
		}

		// 根据状态码选择日志级别
		var level slog.Level
		var msg string
		switch {
		case status >= 500:
			level = slog.LevelError
			msg = "Server error"
			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
			}
		case status >= 400:
			level = slog.LevelWarn
			msg = "Client error"
			if err != nil {
				attrs = append(attrs, slog.String("error", err.Error()))
			}
		case status >= 300:
			level = slog.LevelInfo
			msg = "Redirection"
		default:
			level = slog.LevelInfo
			msg = "Success"
		}

		// 记录日志
		logger.LogAttrs(context.Background(), level, msg, attrs...)

		return err
	}
}

var (
	// Log 保留 zap logger 用于直接使用
	Log *zap.Logger
	// SlogLogger 提供 slog.Logger 用于 Echo v5 集成
	SlogLogger *slog.Logger
)

func InitLogger() {
	// 0. 确保日志目录存在（lumberjack 不会自动创建目录，缺失会导致只写控制台）
	logPath := "./logs/app.log"
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		// 无法创建目录时退化为仅控制台，避免启动失败
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		core := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(os.Stdout), zap.InfoLevel)
		Log = zap.New(core, zap.AddCaller())
		SlogLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
		return
	}
	// 1. 定义日志文件的滚动规则
	w := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10,   // 每个文件最大 10MB
		MaxBackups: 5,    // 保留最近 5 个旧文件
		MaxAge:     30,   // 保留最近 30 天的日志
		Compress:   true, // 是否压缩旧文件
	})

	// 2. 设置编码配置（JSON 格式适合生产，Console 格式适合开发）
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // 使用可读的时间格式

	// 3. 创建 Core
	// 如果你想同时输出到控制台和文件，可以使用 NewTee
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), w, zap.InfoLevel),
		zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(os.Stdout), zap.InfoLevel),
	)

	// 4. 生成 Logger
	Log = zap.New(core, zap.AddCaller())

	// 5. 初始化 SlogLogger，供 Echo ec.Logger 使用；否则 ec.Logger 为 nil，框架会用默认 logger 只打控制台
	SlogLogger = slog.New(NewZapHandler(Log, slog.LevelInfo))
}

// ZapHandler 实现 slog.Handler 接口，将 slog 调用转换为 zap
type ZapHandler struct {
	logger *zap.Logger
	level  slog.Level
}

// NewZapHandler 创建新的 ZapHandler
func NewZapHandler(logger *zap.Logger, level slog.Level) *ZapHandler {
	return &ZapHandler{
		logger: logger,
		level:  level,
	}
}

// Enabled 检查指定级别是否启用
func (h *ZapHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle 处理日志记录
func (h *ZapHandler) Handle(ctx context.Context, record slog.Record) error {
	fields := make([]zap.Field, 0, record.NumAttrs())
	record.Attrs(func(a slog.Attr) bool {
		fields = append(fields, zap.Any(a.Key, a.Value.Any()))
		return true
	})

	// 添加时间戳
	fields = append(fields, zap.Time("time", record.Time))

	// 根据级别记录日志
	switch record.Level {
	case slog.LevelDebug:
		h.logger.Debug(record.Message, fields...)
	case slog.LevelInfo:
		h.logger.Info(record.Message, fields...)
	case slog.LevelWarn:
		h.logger.Warn(record.Message, fields...)
	case slog.LevelError:
		h.logger.Error(record.Message, fields...)
	default:
		h.logger.Info(record.Message, fields...)
	}

	return nil
}

// WithAttrs 返回带有附加属性的新 Handler
func (h *ZapHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	fields := make([]zap.Field, len(attrs))
	for i, attr := range attrs {
		fields[i] = zap.Any(attr.Key, attr.Value.Any())
	}
	return &ZapHandler{
		logger: h.logger.With(fields...),
		level:  h.level,
	}
}

// WithGroup 返回带有组名的新 Handler
func (h *ZapHandler) WithGroup(name string) slog.Handler {
	return &ZapHandler{
		logger: h.logger.Named(name),
		level:  h.level,
	}
}

// 提供便捷的日志方法（保持向后兼容）
func Debug(msg string, fields ...zap.Field) {
	Log.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	Log.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Log.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Log.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	Log.Fatal(msg, fields...)
}

// RequestLoggerWithZap 使用 Echo v5 官方 RequestLogger 配置，集成 zap logger
// 这是推荐的方式，功能更全面、性能更好、维护成本更低
func RequestLoggerWithZap() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		// 配置需要记录的字段
		LogLatency:       true,  // 延迟时间
		LogRemoteIP:      true,  // 远程 IP
		LogHost:          true,  // 主机名
		LogMethod:        true,  // HTTP 方法
		LogURI:           true,  // 完整 URI（包含查询参数）
		LogURIPath:       false, // URI 路径（不含查询参数）
		LogRoutePath:     false, // 路由路径（如 /user/:id）
		LogRequestID:     true,  // 请求 ID
		LogUserAgent:     true,  // User Agent
		LogStatus:        true,  // 响应状态码
		LogResponseSize:  true,  // 响应大小
		LogContentLength: false, // 请求体大小（可能被伪造）
		LogProtocol:      false, // HTTP 协议版本
		LogReferer:       false, // Referer

		// 错误处理：当 handler 返回错误时，调用全局错误处理器
		// 这样可以确保状态码正确（错误处理器可能修改状态码）
		HandleError: true,

		// 日志记录回调函数
		LogValuesFunc: func(c *echo.Context, v middleware.RequestLoggerValues) error {
			// 使用项目中的 zap logger
			zapLogger := Log

			// 构建 zap fields
			fields := []zap.Field{
				zap.String("method", v.Method),
				zap.String("uri", v.URI),
				zap.Int("status", v.Status),
				zap.Duration("latency", v.Latency),
				zap.String("host", v.Host),
				zap.Int64("bytes_out", v.ResponseSize),
				zap.String("user_agent", v.UserAgent),
				zap.String("remote_ip", v.RemoteIP),
			}

			// 添加请求 ID（如果存在）
			if v.RequestID != "" {
				fields = append(fields, zap.String("request_id", v.RequestID))
			}

			// 根据状态码和错误选择日志级别
			if v.Error != nil {
				// 有错误时记录错误信息
				fields = append(fields, zap.Error(v.Error))
				zapLogger.Error("REQUEST_ERROR", fields...)
			} else {
				// 根据状态码选择级别
				switch {
				case v.Status >= 500:
					zapLogger.Error("Server error", fields...)
				case v.Status >= 400:
					zapLogger.Warn("Client error", fields...)
				case v.Status >= 300:
					zapLogger.Info("Redirection", fields...)
				default:
					zapLogger.Info("Success", fields...)
				}
			}

			return nil
		},
	})
}
