package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type JWTConfig struct {
	Secret   string        `mapstructure:"secret"`
	Duration time.Duration `mapstructure:"duration"`
}
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`    // 日志文件路径
	MaxSize    int    `mapstructure:"max_size"`    // 每个文件最大 10MB
	MaxBackups int    `mapstructure:"max_backups"` // 保留最近 5 个旧文件
	MaxAge     int    `mapstructure:"max_age"`     // 保留最近 30 天的日志
	Compress   bool   `mapstructure:"compress" `
}
type Config struct {
	Server *ServerInfo `mapstructure:"server"`
	Log    *LogConfig  `mapstructure:"log"`
	JWT    *JWTConfig  `mapstructure:"jwt"`
}
type ServerInfo struct {
	Address           string        `mapstructure:"address"`
	Port              string        `mapstructure:"port"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	MaxHeaderBytes    int           `mapstructure:"max_header_bytes"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
	// 中间件：请求体大小限制（字节）
	BodyLimit int64 `mapstructure:"body_limit"`
	// 限流：每秒请求数、突发容量
	RateLimitRate  float64 `mapstructure:"rate_limit_rate"`
	RateLimitBurst int     `mapstructure:"rate_limit_burst"`
}
type DatabaseConfig struct {
	Driver       string `mapstructure:"driver"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"ssl_mode"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
}

func NewConfig(filePath string) (*Config, error) {
	fmt.Println("正在加载路径:", filePath)

	v := viper.New() // 建议使用局部实例，避免全局污染
	v.SetConfigFile(filePath)
	v.SetConfigType("yaml") // 明确指定格式

	v.SetEnvPrefix("maotuan")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 1. 必须先读取文件
	if err := v.ReadInConfig(); err != nil {
		// 如果是因为找不到文件，可以打印更清晰的错误
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("配置文件未找到: %s", filePath)
		}
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 2. 初始化结构体
	conf := &Config{}

	// 3. 解析到结构体
	if err := v.Unmarshal(conf); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	return conf, nil
}
