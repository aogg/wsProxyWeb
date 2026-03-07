// 配置初始化逻辑
package logic

import "github.com/spf13/viper"

// SetDefaultConfig 设置默认配置值（从server.yaml读取）
func SetDefaultConfig() {
	// 默认配置已在server.yaml中定义，这里不再硬编码
	// viper会在ReadInConfig时自动读取yaml中的值
}

// BindEnvVars 绑定环境变量到配置键
func BindEnvVars() {
	// 服务器配置
	viper.BindEnv("server.host", "WS_PROXY_SERVER_HOST")
	viper.BindEnv("server.port", "WS_PROXY_SERVER_PORT")

	// 日志配置
	viper.BindEnv("log.enabled", "WS_PROXY_LOG_ENABLED")
	viper.BindEnv("log.level", "WS_PROXY_LOG_LEVEL")
	viper.BindEnv("log.logDir", "WS_PROXY_LOG_LOGDIR")
	viper.BindEnv("log.console", "WS_PROXY_LOG_CONSOLE")
	viper.BindEnv("log.file", "WS_PROXY_LOG_FILE")
	viper.BindEnv("log.colorConsole", "WS_PROXY_LOG_COLORCONSOLE")
	viper.BindEnv("log.maxAgeDays", "WS_PROXY_LOG_MAXAGEDAYS")

	// 认证配置
	viper.BindEnv("auth.enabled", "WS_PROXY_AUTH_ENABLED")
	viper.BindEnv("auth.adminUsername", "WS_PROXY_AUTH_ADMINUSERNAME")
	viper.BindEnv("auth.adminPassword", "WS_PROXY_AUTH_ADMINPASSWORD")
	viper.BindEnv("auth.userDataDir", "WS_PROXY_AUTH_USERDATADIR")

	// 加密配置
	viper.BindEnv("crypto.enabled", "WS_PROXY_CRYPTO_ENABLED")
	viper.BindEnv("crypto.key", "WS_PROXY_CRYPTO_KEY")
	viper.BindEnv("crypto.algorithm", "WS_PROXY_CRYPTO_ALGORITHM")

	// 压缩配置
	viper.BindEnv("compress.enabled", "WS_PROXY_COMPRESS_ENABLED")
	viper.BindEnv("compress.level", "WS_PROXY_COMPRESS_LEVEL")
	viper.BindEnv("compress.algorithm", "WS_PROXY_COMPRESS_ALGORITHM")

	// 安全配置
	viper.BindEnv("security.enabled", "WS_PROXY_SECURITY_ENABLED")
	viper.BindEnv("security.maxConnections", "WS_PROXY_SECURITY_MAXCONNECTIONS")
	viper.BindEnv("security.maxMessageBytes", "WS_PROXY_SECURITY_MAXMESSAGEBYTES")
	viper.BindEnv("security.maxRequestBodyBytes", "WS_PROXY_SECURITY_MAXREQUESTBODYBYTES")
	viper.BindEnv("security.rateLimitPerSecond", "WS_PROXY_SECURITY_RATELIMITPERSECOND")
	viper.BindEnv("security.rateBurst", "WS_PROXY_SECURITY_RATEBURST")

	// HTTP客户端配置
	viper.BindEnv("http.timeoutSeconds", "WS_PROXY_HTTP_TIMEOUTSECONDS")
	viper.BindEnv("http.maxIdleConns", "WS_PROXY_HTTP_MAXIDLECONNS")
	viper.BindEnv("http.maxIdleConnsPerHost", "WS_PROXY_HTTP_MAXIDLECONNSPERHOST")
	viper.BindEnv("http.idleConnTimeoutSeconds", "WS_PROXY_HTTP_IDLECONNTIMEOUTSECONDS")
	viper.BindEnv("http.tlsHandshakeTimeoutSeconds", "WS_PROXY_HTTP_TLSHANDSHAKETIMEOUTSECONDS")
	viper.BindEnv("http.expectContinueTimeoutSeconds", "WS_PROXY_HTTP_EXPECTCONTINUETIMEOUTSECONDS")
	viper.BindEnv("http.proxyEnabled", "WS_PROXY_HTTP_PROXYENABLED")
	viper.BindEnv("http.proxyURL", "WS_PROXY_HTTP_PROXYURL")

	// 性能配置
	viper.BindEnv("performance.workerPoolSize", "WS_PROXY_PERFORMANCE_WORKERPOOLSIZE")
	viper.BindEnv("performance.requestQueueSize", "WS_PROXY_PERFORMANCE_REQUESTQUEUESIZE")
	viper.BindEnv("performance.maxConcurrentConns", "WS_PROXY_PERFORMANCE_MAXCONCURRENTCONNS")
	viper.BindEnv("performance.bufferPoolSize", "WS_PROXY_PERFORMANCE_BUFFERPOOLSIZE")
	viper.BindEnv("performance.chunkSize", "WS_PROXY_PERFORMANCE_CHUNKSIZE")
	viper.BindEnv("performance.enableMetrics", "WS_PROXY_PERFORMANCE_ENABLEMETRICS")
	viper.BindEnv("performance.metricsIntervalSec", "WS_PROXY_PERFORMANCE_METRICSINTERVALSEC")
}
