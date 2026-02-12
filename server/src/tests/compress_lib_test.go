package tests

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/golang/snappy"
	"wsProxyWeb/server/src/libs"
)

// 测试gzip压缩解压
func TestCompressLib_Gzip_CompressDecompress(t *testing.T) {
	config := &libs.CompressConfig{
		Enabled:   true,
		Level:     6,
		Algorithm: "gzip",
	}

	compressLib, err := libs.NewCompressLib(config)
	if err != nil {
		t.Fatalf("创建CompressLib失败: %v", err)
	}

	// 测试数据
	originalData := []byte("Hello, World! 这是一个测试消息，用于测试gzip压缩功能。" +
		"重复的内容可以提高压缩率。重复的内容可以提高压缩率。")

	// 压缩
	compressed, err := compressLib.Compress(originalData)
	if err != nil {
		t.Fatalf("压缩失败: %v", err)
	}

	// 验证压缩后数据变小
	if len(compressed) >= len(originalData) {
		t.Logf("警告: 压缩后数据没有变小，原始: %d, 压缩后: %d", len(originalData), len(compressed))
	}

	// 解压
	decompressed, err := compressLib.Decompress(compressed)
	if err != nil {
		t.Fatalf("解压失败: %v", err)
	}

	// 验证解压结果
	if !bytes.Equal(decompressed, originalData) {
		t.Errorf("解压结果不匹配")
	}
}

// 测试snappy压缩解压
func TestCompressLib_Snappy_CompressDecompress(t *testing.T) {
	config := &libs.CompressConfig{
		Enabled:   true,
		Level:     6,
		Algorithm: "snappy",
	}

	compressLib, err := libs.NewCompressLib(config)
	if err != nil {
		t.Fatalf("创建CompressLib失败: %v", err)
	}

	// 测试数据
	originalData := []byte("Hello, World! 这是一个测试消息，用于测试snappy压缩功能。" +
		"Snappy注重速度而非压缩率。")

	// 压缩
	compressed, err := compressLib.Compress(originalData)
	if err != nil {
		t.Fatalf("压缩失败: %v", err)
	}

	// 解压
	decompressed, err := compressLib.Decompress(compressed)
	if err != nil {
		t.Fatalf("解压失败: %v", err)
	}

	// 验证解压结果
	if !bytes.Equal(decompressed, originalData) {
		t.Errorf("解压结果不匹配")
	}
}

// 测试未启用压缩时直接返回原文
func TestCompressLib_Disabled(t *testing.T) {
	config := &libs.CompressConfig{
		Enabled:   false,
		Level:     6,
		Algorithm: "gzip",
	}

	compressLib, err := libs.NewCompressLib(config)
	if err != nil {
		t.Fatalf("创建CompressLib失败: %v", err)
	}

	if compressLib.IsEnabled() {
		t.Error("未启用压缩时，IsEnabled应该返回false")
	}

	originalData := []byte("测试数据")

	// 压缩应该直接返回原文
	compressed, err := compressLib.Compress(originalData)
	if err != nil {
		t.Fatalf("压缩失败: %v", err)
	}
	if !bytes.Equal(compressed, originalData) {
		t.Errorf("未启用压缩时，压缩应该返回原文")
	}

	// 解压应该直接返回原文
	decompressed, err := compressLib.Decompress(originalData)
	if err != nil {
		t.Fatalf("解压失败: %v", err)
	}
	if !bytes.Equal(decompressed, originalData) {
		t.Errorf("未启用压缩时，解压应该返回原文")
	}
}

// 测试压缩级别范围验证
func TestCompressLib_InvalidLevel(t *testing.T) {
	// 测试级别过低
	config := &libs.CompressConfig{
		Enabled:   true,
		Level:     0,
		Algorithm: "gzip",
	}

	_, err := libs.NewCompressLib(config)
	if err == nil {
		t.Error("压缩级别0应该返回错误")
	}

	// 测试级别过高
	config.Level = 10
	_, err = libs.NewCompressLib(config)
	if err == nil {
		t.Error("压缩级别10应该返回错误")
	}
}

// 测试不支持的算法
func TestCompressLib_UnsupportedAlgorithm(t *testing.T) {
	config := &libs.CompressConfig{
		Enabled:   true,
		Level:     6,
		Algorithm: "invalid_algorithm",
	}

	_, err := libs.NewCompressLib(config)
	if err == nil {
		t.Error("不支持的算法应该返回错误")
	}
}

// 测试不同压缩级别
func TestCompressLib_DifferentLevels(t *testing.T) {
	originalData := make([]byte, 1024)
	for i := range originalData {
		originalData[i] = byte(i % 10) // 重复模式，易于压缩
	}

	levels := []int{1, 3, 6, 9}

	for _, level := range levels {
		config := &libs.CompressConfig{
			Enabled:   true,
			Level:     level,
			Algorithm: "gzip",
		}

		compressLib, err := libs.NewCompressLib(config)
		if err != nil {
			t.Fatalf("创建CompressLib失败 (level=%d): %v", level, err)
		}

		compressed, err := compressLib.Compress(originalData)
		if err != nil {
			t.Fatalf("压缩失败 (level=%d): %v", level, err)
		}

		decompressed, err := compressLib.Decompress(compressed)
		if err != nil {
			t.Fatalf("解压失败 (level=%d): %v", level, err)
		}

		if !bytes.Equal(decompressed, originalData) {
			t.Errorf("解压结果不匹配 (level=%d)", level)
		}

		t.Logf("Level %d: 原始大小=%d, 压缩后大小=%d, 压缩率=%.2f%%",
			level, len(originalData), len(compressed),
			float64(len(compressed))/float64(len(originalData))*100)
	}
}

// 测试空数据压缩
func TestCompressLib_EmptyData(t *testing.T) {
	config := &libs.CompressConfig{
		Enabled:   true,
		Level:     6,
		Algorithm: "gzip",
	}

	compressLib, err := libs.NewCompressLib(config)
	if err != nil {
		t.Fatalf("创建CompressLib失败: %v", err)
	}

	// 压缩空数据
	originalData := []byte{}
	compressed, err := compressLib.Compress(originalData)
	if err != nil {
		t.Fatalf("压缩空数据失败: %v", err)
	}

	// 解压
	decompressed, err := compressLib.Decompress(compressed)
	if err != nil {
		t.Fatalf("解压失败: %v", err)
	}

	if len(decompressed) != 0 {
		t.Errorf("解压空数据应该返回空数据")
	}
}

// 测试大数据压缩
func TestCompressLib_LargeData(t *testing.T) {
	config := &libs.CompressConfig{
		Enabled:   true,
		Level:     6,
		Algorithm: "gzip",
	}

	compressLib, err := libs.NewCompressLib(config)
	if err != nil {
		t.Fatalf("创建CompressLib失败: %v", err)
	}

	// 生成1MB的测试数据
	originalData := make([]byte, 1024*1024)
	for i := range originalData {
		originalData[i] = byte(i % 256)
	}

	// 压缩
	compressed, err := compressLib.Compress(originalData)
	if err != nil {
		t.Fatalf("压缩大数据失败: %v", err)
	}

	t.Logf("大数据压缩: 原始=%d, 压缩后=%d, 压缩率=%.2f%%",
		len(originalData), len(compressed),
		float64(len(compressed))/float64(len(originalData))*100)

	// 解压
	decompressed, err := compressLib.Decompress(compressed)
	if err != nil {
		t.Fatalf("解压大数据失败: %v", err)
	}

	// 验证数据
	if !bytes.Equal(decompressed, originalData) {
		t.Errorf("解压大数据结果不匹配")
	}
}

// 测试解压无效数据
func TestCompressLib_DecompressInvalidData(t *testing.T) {
	config := &libs.CompressConfig{
		Enabled:   true,
		Level:     6,
		Algorithm: "gzip",
	}

	compressLib, err := libs.NewCompressLib(config)
	if err != nil {
		t.Fatalf("创建CompressLib失败: %v", err)
	}

	// 解压无效的gzip数据
	invalidData := []byte("这不是有效的gzip数据")
	_, err = compressLib.Decompress(invalidData)
	if err == nil {
		t.Error("解压无效数据应该失败")
	}
}

// 测试解压无效snappy数据
func TestCompressLib_DecompressInvalidSnappyData(t *testing.T) {
	config := &libs.CompressConfig{
		Enabled:   true,
		Level:     6,
		Algorithm: "snappy",
	}

	compressLib, err := libs.NewCompressLib(config)
	if err != nil {
		t.Fatalf("创建CompressLib失败: %v", err)
	}

	// 解压无效的snappy数据
	invalidData := []byte("这不是有效的snappy数据")
	_, err = compressLib.Decompress(invalidData)
	if err == nil {
		t.Error("解压无效snappy数据应该失败")
	}
}

// 测试与标准库兼容性 - gzip
func TestCompressLib_GzipCompatibility(t *testing.T) {
	config := &libs.CompressConfig{
		Enabled:   true,
		Level:     6,
		Algorithm: "gzip",
	}

	compressLib, err := libs.NewCompressLib(config)
	if err != nil {
		t.Fatalf("创建CompressLib失败: %v", err)
	}

	originalData := []byte("测试gzip兼容性数据")

	// 使用我们的库压缩
	compressed, err := compressLib.Compress(originalData)
	if err != nil {
		t.Fatalf("压缩失败: %v", err)
	}

	// 使用标准库解压
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		t.Fatalf("标准库创建reader失败: %v", err)
	}
	defer reader.Close()

	decompressed := make([]byte, len(originalData))
	n, err := reader.Read(decompressed)
	if err != nil {
		t.Fatalf("标准库解压失败: %v", err)
	}

	if !bytes.Equal(decompressed[:n], originalData) {
		t.Errorf("与标准库不兼容")
	}
}

// 测试与标准库兼容性 - snappy
func TestCompressLib_SnappyCompatibility(t *testing.T) {
	config := &libs.CompressConfig{
		Enabled:   true,
		Level:     6,
		Algorithm: "snappy",
	}

	compressLib, err := libs.NewCompressLib(config)
	if err != nil {
		t.Fatalf("创建CompressLib失败: %v", err)
	}

	originalData := []byte("测试snappy兼容性数据")

	// 使用我们的库压缩
	compressed, err := compressLib.Compress(originalData)
	if err != nil {
		t.Fatalf("压缩失败: %v", err)
	}

	// 使用标准库解压
	decompressed, err := snappy.Decode(nil, compressed)
	if err != nil {
		t.Fatalf("标准库解压失败: %v", err)
	}

	if !bytes.Equal(decompressed, originalData) {
		t.Errorf("与标准库不兼容")
	}
}

// 测试重复压缩（已压缩数据再压缩）
func TestCompressLib_CompressTwice(t *testing.T) {
	config := &libs.CompressConfig{
		Enabled:   true,
		Level:     6,
		Algorithm: "gzip",
	}

	compressLib, err := libs.NewCompressLib(config)
	if err != nil {
		t.Fatalf("创建CompressLib失败: %v", err)
	}

	originalData := []byte("测试重复压缩")

	// 第一次压缩
	compressed1, err := compressLib.Compress(originalData)
	if err != nil {
		t.Fatalf("第一次压缩失败: %v", err)
	}

	// 第二次压缩已压缩的数据
	compressed2, err := compressLib.Compress(compressed1)
	if err != nil {
		t.Fatalf("第二次压缩失败: %v", err)
	}

	// 第一次解压
	decompressed1, err := compressLib.Decompress(compressed2)
	if err != nil {
		t.Fatalf("第一次解压失败: %v", err)
	}

	// 第二次解压
	decompressed2, err := compressLib.Decompress(decompressed1)
	if err != nil {
		t.Fatalf("第二次解压失败: %v", err)
	}

	// 验证最终结果
	if !bytes.Equal(decompressed2, originalData) {
		t.Errorf("重复压缩后解压结果不匹配")
	}
}
