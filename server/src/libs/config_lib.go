// 配置加载工具库
package libs

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
)

// ServerConfig 服务端配置结构
type ServerConfig struct {
	Server      ServerConfigServer `mapstructure:"server"`
	Log         LogConfig          `mapstructure:"log"`
	Auth        AuthConfig         `mapstructure:"auth"`
	Crypto      CryptoConfig       `mapstructure:"crypto"`
	Compress    CompressConfig     `mapstructure:"compress"`
	Security    SecurityConfig     `mapstructure:"security"`
	HTTP        HTTPConfig         `mapstructure:"http"`
	Performance PerformanceConfig  `mapstructure:"performance"`
}

// AuthConfig 认证配置
type AuthConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	AdminUsername string `mapstructure:"adminUsername"`
	AdminPassword string `mapstructure:"adminPassword"`
	UserDataDir   string `mapstructure:"userDataDir"`
}

// ServerConfigServer 服务器配置
type ServerConfigServer struct {
	Host string `mapstructure:"host"`
	Port string `mapstructure:"port"`
}

// CryptoConfig 加密配置
type CryptoConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Key       string `mapstructure:"key"`       // 加密密钥（32字节，Base64编码）
	Algorithm string `mapstructure:"algorithm"` // 加密算法：aes256gcm 或 chacha20poly1305
}

// CompressConfig 压缩配置
type CompressConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Level     int    `mapstructure:"level"`     // 压缩级别：1-9，默认6
	Algorithm string `mapstructure:"algorithm"` // 压缩算法：gzip 或 snappy
}

// SecurityConfig 安全控制配置
type SecurityConfig struct {
	Enabled             bool `mapstructure:"enabled"`
	MaxConnections      int  `mapstructure:"maxConnections"`      // 最大连接数（0表示不限制）
	MaxMessageBytes     int  `mapstructure:"maxMessageBytes"`     // 单条消息最大字节数（0表示不限制，指解密前rawData大小）
	MaxRequestBodyBytes int  `mapstructure:"maxRequestBodyBytes"` // 请求体最大字节数（0表示不限制，指解码后的字节数）

	RateLimitPerSecond float64 `mapstructure:"rateLimitPerSecond"` // 每秒允许的请求数（<=0表示不限制）
	RateBurst          int     `mapstructure:"rateBurst"`          // 突发容量（<=0时将按rate自动给默认值）

	AllowIPs     []string `mapstructure:"allowIPs"`     // 允许IP列表，支持单IP或CIDR（为空表示不限制）
	DenyIPs      []string `mapstructure:"denyIPs"`      // 拒绝IP列表，支持单IP或CIDR
	AllowDomains []string `mapstructure:"allowDomains"` // 允许域名列表，支持*.example.com（为空表示不限制）
	DenyDomains  []string `mapstructure:"denyDomains"`  // 拒绝域名列表，支持*.example.com
}

// HTTPConfig HTTP客户端配置（用于服务端转发请求）
type HTTPConfig struct {
	TimeoutSeconds               int    `mapstructure:"timeoutSeconds"`               // 请求总超时秒数
	MaxIdleConns                 int    `mapstructure:"maxIdleConns"`                 // 全局最大空闲连接
	MaxIdleConnsPerHost          int    `mapstructure:"maxIdleConnsPerHost"`          // 单host最大空闲连接
	IdleConnTimeoutSeconds       int    `mapstructure:"idleConnTimeoutSeconds"`       // 空闲连接超时秒数
	TLSHandshakeTimeoutSeconds   int    `mapstructure:"tlsHandshakeTimeoutSeconds"`   // TLS握手超时秒数
	ExpectContinueTimeoutSeconds int    `mapstructure:"expectContinueTimeoutSeconds"` // Expect: 100-continue 超时秒数
	ProxyEnabled                 bool   `mapstructure:"proxyEnabled"`                 // 是否启用正向代理
	ProxyURL                     string `mapstructure:"proxyURL"`                     // 正向代理地址
}

var globalConfig *ServerConfig

// LoadConfig 加载配置文件
func LoadConfig(configPath string) (*ServerConfig, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}

	// 设置配置文件路径
	if configPath == "" {
		// 默认配置文件路径
		configPath = "src/configs/server.yaml"
	}

	// 获取配置文件的绝对路径
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("获取配置文件绝对路径失败: %v", err)
	}

	// 设置viper配置
	viper.SetConfigFile(absPath)
	viper.SetConfigType("yaml")

	// 设置默认值
	setDefaultConfig()

	// 启用环境变量支持
	viper.SetEnvPrefix("WS_PROXY")
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		Warn("读取配置文件失败: %v，使用默认配置", err)
	}

	// 解析配置到结构体
	config := &ServerConfig{}
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 验证配置
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	globalConfig = config
	Info("配置文件加载成功: %s", absPath)
	return config, nil
}

// GetConfig 获取全局配置
func GetConfig() *ServerConfig {
	if globalConfig == nil {
		// 尝试加载默认配置
		config, err := LoadConfig("")
		if err != nil {
			Warn("加载默认配置失败: %v，使用空配置", err)
			globalConfig = &ServerConfig{}
		} else {
			globalConfig = config
		}
	}
	return globalConfig
}

// setDefaultConfig 设置默认配置值
func setDefaultConfig() {
	// 服务器默认配置
	viper.SetDefault("server.port", "8080")

	// 日志默认配置
	viper.SetDefault("log.enabled", true)
	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.logDir", "runtime/server/logs")
	viper.SetDefault("log.console", true)
	viper.SetDefault("log.file", true)
	viper.SetDefault("log.colorConsole", true)
	viper.SetDefault("log.maxAgeDays", 30)

	// 认证默认配置
	viper.SetDefault("auth.enabled", true)
	viper.SetDefault("auth.adminUsername", "admin")
	viper.SetDefault("auth.adminPassword", "admin123")
	viper.SetDefault("auth.userDataDir", "runtime/server/data")

	// 加密默认配置
	viper.SetDefault("crypto.enabled", false)
	viper.SetDefault("crypto.key", "")
	viper.SetDefault("crypto.algorithm", "aes256gcm")

	// 压缩默认配置
	viper.SetDefault("compress.enabled", false)
	viper.SetDefault("compress.level", 6)
	viper.SetDefault("compress.algorithm", "gzip")

	// 安全控制默认配置（默认启用，但规则为空时等于“全放行”，仅做基础限流/限大小/限连接）
	viper.SetDefault("security.enabled", true)
	viper.SetDefault("security.maxConnections", 50)
	viper.SetDefault("security.maxMessageBytes", 2*1024*1024)     // 2MB
	viper.SetDefault("security.maxRequestBodyBytes", 5*1024*1024) // 5MB
	viper.SetDefault("security.rateLimitPerSecond", 50.0)
	viper.SetDefault("security.rateBurst", 100)
	viper.SetDefault("security.allowIPs", []string{})
	viper.SetDefault("security.denyIPs", []string{})
	viper.SetDefault("security.allowDomains", []string{})
	viper.SetDefault("security.denyDomains", []string{})

	// HTTP客户端默认配置（连接池复用）
	viper.SetDefault("http.timeoutSeconds", 30)
	viper.SetDefault("http.maxIdleConns", 200)
	viper.SetDefault("http.maxIdleConnsPerHost", 50)
	viper.SetDefault("http.idleConnTimeoutSeconds", 90)
	viper.SetDefault("http.tlsHandshakeTimeoutSeconds", 10)
	viper.SetDefault("http.expectContinueTimeoutSeconds", 1)
	viper.SetDefault("http.proxyEnabled", false)
	viper.SetDefault("http.proxyURL", "")

	// 性能优化默认配置
	viper.SetDefault("performance.workerPoolSize", 0) // 默认不使用worker池
	viper.SetDefault("performance.requestQueueSize", 1000)
	viper.SetDefault("performance.maxConcurrentConns", 0) // 不限制
	viper.SetDefault("performance.bufferPoolSize", 100)
	viper.SetDefault("performance.chunkSize", 65536) // 64KB
	viper.SetDefault("performance.enableMetrics", true)
	viper.SetDefault("performance.metricsIntervalSec", 30)
}

// validateConfig 验证配置
func validateConfig(config *ServerConfig) error {
	// 验证加密配置
	if config.Crypto.Enabled {
		if config.Crypto.Key == "" {
			return fmt.Errorf("加密已启用，但未配置密钥")
		}
		if config.Crypto.Algorithm != "aes256gcm" && config.Crypto.Algorithm != "chacha20poly1305" {
			return fmt.Errorf("不支持的加密算法: %s，支持: aes256gcm, chacha20poly1305", config.Crypto.Algorithm)
		}
	}

	// 验证压缩配置
	if config.Compress.Enabled {
		if config.Compress.Level < 1 || config.Compress.Level > 9 {
			return fmt.Errorf("压缩级别必须在1-9之间，当前值: %d", config.Compress.Level)
		}
		if config.Compress.Algorithm != "gzip" && config.Compress.Algorithm != "snappy" {
			return fmt.Errorf("不支持的压缩算法: %s，支持: gzip, snappy", config.Compress.Algorithm)
		}
	}

	// 验证安全控制配置
	if config.Security.MaxConnections < 0 {
		return fmt.Errorf("maxConnections不能小于0，当前值: %d", config.Security.MaxConnections)
	}
	if config.Security.MaxMessageBytes < 0 {
		return fmt.Errorf("maxMessageBytes不能小于0，当前值: %d", config.Security.MaxMessageBytes)
	}
	if config.Security.MaxRequestBodyBytes < 0 {
		return fmt.Errorf("maxRequestBodyBytes不能小于0，当前值: %d", config.Security.MaxRequestBodyBytes)
	}
	if config.Security.RateLimitPerSecond < 0 {
		return fmt.Errorf("rateLimitPerSecond不能小于0，当前值: %f", config.Security.RateLimitPerSecond)
	}
	if config.Security.RateBurst < 0 {
		return fmt.Errorf("rateBurst不能小于0，当前值: %d", config.Security.RateBurst)
	}

	// 验证HTTP客户端配置
	if config.HTTP.TimeoutSeconds <= 0 {
		return fmt.Errorf("http.timeoutSeconds必须大于0，当前值: %d", config.HTTP.TimeoutSeconds)
	}
	if config.HTTP.MaxIdleConns < 0 {
		return fmt.Errorf("http.maxIdleConns不能小于0，当前值: %d", config.HTTP.MaxIdleConns)
	}
	if config.HTTP.MaxIdleConnsPerHost < 0 {
		return fmt.Errorf("http.maxIdleConnsPerHost不能小于0，当前值: %d", config.HTTP.MaxIdleConnsPerHost)
	}
	if config.HTTP.IdleConnTimeoutSeconds < 0 {
		return fmt.Errorf("http.idleConnTimeoutSeconds不能小于0，当前值: %d", config.HTTP.IdleConnTimeoutSeconds)
	}
	if config.HTTP.TLSHandshakeTimeoutSeconds < 0 {
		return fmt.Errorf("http.tlsHandshakeTimeoutSeconds不能小于0，当前值: %d", config.HTTP.TLSHandshakeTimeoutSeconds)
	}
	if config.HTTP.ExpectContinueTimeoutSeconds < 0 {
		return fmt.Errorf("http.expectContinueTimeoutSeconds不能小于0，当前值: %d", config.HTTP.ExpectContinueTimeoutSeconds)
	}

	return nil
}
