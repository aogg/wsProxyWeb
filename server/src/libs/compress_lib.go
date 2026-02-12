// 压缩工具库
package libs

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"

	"github.com/golang/snappy"
)

// CompressLib 压缩库结构
type CompressLib struct {
	enabled   bool
	level     int
	algorithm string
}

// NewCompressLib 创建新的压缩库实例
func NewCompressLib(config *CompressConfig) (*CompressLib, error) {
	lib := &CompressLib{
		enabled:   config.Enabled,
		level:     config.Level,
		algorithm: config.Algorithm,
	}

	if !config.Enabled {
		return lib, nil
	}

	// 验证压缩级别
	if config.Level < 1 || config.Level > 9 {
		return nil, fmt.Errorf("压缩级别必须在1-9之间，当前值: %d", config.Level)
	}

	// 验证算法
	if config.Algorithm != "gzip" && config.Algorithm != "snappy" {
		return nil, fmt.Errorf("不支持的压缩算法: %s，支持: gzip, snappy", config.Algorithm)
	}

	return lib, nil
}

// Compress 压缩数据
func (c *CompressLib) Compress(data []byte) ([]byte, error) {
	if !c.enabled {
		return data, nil
	}

	switch c.algorithm {
	case "gzip":
		return c.compressGzip(data)
	case "snappy":
		return c.compressSnappy(data)
	default:
		return nil, fmt.Errorf("不支持的压缩算法: %s", c.algorithm)
	}
}

// Decompress 解压数据
func (c *CompressLib) Decompress(data []byte) ([]byte, error) {
	if !c.enabled {
		return data, nil
	}

	switch c.algorithm {
	case "gzip":
		return c.decompressGzip(data)
	case "snappy":
		return c.decompressSnappy(data)
	default:
		return nil, fmt.Errorf("不支持的压缩算法: %s", c.algorithm)
	}
}

// compressGzip 使用gzip压缩
func (c *CompressLib) compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	// 创建gzip writer
	writer, err := gzip.NewWriterLevel(&buf, c.level)
	if err != nil {
		return nil, fmt.Errorf("创建gzip writer失败: %v", err)
	}

	// 写入数据
	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return nil, fmt.Errorf("写入压缩数据失败: %v", err)
	}

	// 关闭writer（必须调用，否则数据不完整）
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭gzip writer失败: %v", err)
	}

	return buf.Bytes(), nil
}

// decompressGzip 使用gzip解压
func (c *CompressLib) decompressGzip(data []byte) ([]byte, error) {
	// 创建gzip reader
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("创建gzip reader失败: %v", err)
	}
	defer reader.Close()

	// 读取解压后的数据
	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("读取解压数据失败: %v", err)
	}

	return result, nil
}

// compressSnappy 使用snappy压缩
func (c *CompressLib) compressSnappy(data []byte) ([]byte, error) {
	// snappy压缩（snappy不依赖压缩级别）
	compressed := snappy.Encode(nil, data)
	return compressed, nil
}

// decompressSnappy 使用snappy解压
func (c *CompressLib) decompressSnappy(data []byte) ([]byte, error) {
	// snappy解压
	decompressed, err := snappy.Decode(nil, data)
	if err != nil {
		return nil, fmt.Errorf("snappy解压失败: %v", err)
	}
	return decompressed, nil
}

// IsEnabled 检查压缩是否启用
func (c *CompressLib) IsEnabled() bool {
	return c.enabled
}
