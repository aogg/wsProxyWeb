// 配置加载工具库
package libs

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/spf13/viper"
)

// ServerConfig 服务端配置结构
type ServerConfig struct {
	Server ServerConfigServer `mapstructure:"server"`
	Crypto CryptoConfig       `mapstructure:"crypto"`
	Compress CompressConfig   `mapstructure:"compress"`
}

// ServerConfigServer 服务器配置
type ServerConfigServer struct {
	Port string `mapstructure:"port"`
}

// CryptoConfig 加密配置
type CryptoConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Key     string `mapstructure:"key"`     // 加密密钥（32字节，Base64编码）
	Algorithm string `mapstructure:"algorithm"` // 加密算法：aes256gcm 或 chacha20poly1305
}

// CompressConfig 压缩配置
type CompressConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Level   int    `mapstructure:"level"`   // 压缩级别：1-9，默认6
	Algorithm string `mapstructure:"algorithm"` // 压缩算法：gzip 或 snappy
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

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("警告: 读取配置文件失败: %v，使用默认配置", err)
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
	log.Printf("配置文件加载成功: %s", absPath)
	return config, nil
}

// GetConfig 获取全局配置
func GetConfig() *ServerConfig {
	if globalConfig == nil {
		// 尝试加载默认配置
		config, err := LoadConfig("")
		if err != nil {
			log.Printf("加载默认配置失败: %v，使用空配置", err)
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

	// 加密默认配置
	viper.SetDefault("crypto.enabled", false)
	viper.SetDefault("crypto.key", "")
	viper.SetDefault("crypto.algorithm", "aes256gcm")

	// 压缩默认配置
	viper.SetDefault("compress.enabled", false)
	viper.SetDefault("compress.level", 6)
	viper.SetDefault("compress.algorithm", "gzip")
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

	return nil
}
