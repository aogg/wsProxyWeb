package tests

import (
	"encoding/base64"
	"testing"

	"wsProxyWeb/server/src/libs"
)

// 测试AES-256-GCM加密解密
func TestCryptoLib_AES256GCM_EncryptDecrypt(t *testing.T) {
	// 生成32字节密钥
	key := make([]byte, 32)
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	config := &libs.CryptoConfig{
		Enabled:   true,
		Key:       keyBase64,
		Algorithm: "aes256gcm",
	}

	cryptoLib, err := libs.NewCryptoLib(config)
	if err != nil {
		t.Fatalf("创建CryptoLib失败: %v", err)
	}

	// 测试数据
	plaintext := []byte("Hello, World! 这是一个测试消息。")

	// 加密
	ciphertext, err := cryptoLib.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 验证密文长度（nonce 12字节 + 密文 + tag 16字节）
	if len(ciphertext) <= len(plaintext) {
		t.Errorf("密文长度应该大于明文，明文: %d, 密文: %d", len(plaintext), len(ciphertext))
	}

	// 解密
	decrypted, err := cryptoLib.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	// 验证解密结果
	if string(decrypted) != string(plaintext) {
		t.Errorf("解密结果不匹配，期望: %s, 实际: %s", plaintext, decrypted)
	}
}

// 测试ChaCha20-Poly1305加密解密
func TestCryptoLib_ChaCha20Poly1305_EncryptDecrypt(t *testing.T) {
	// 生成32字节密钥
	key := make([]byte, 32)
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	config := &libs.CryptoConfig{
		Enabled:   true,
		Key:       keyBase64,
		Algorithm: "chacha20poly1305",
	}

	cryptoLib, err := libs.NewCryptoLib(config)
	if err != nil {
		t.Fatalf("创建CryptoLib失败: %v", err)
	}

	// 测试数据
	plaintext := []byte("Hello, ChaCha20! 这是另一个测试消息。")

	// 加密
	ciphertext, err := cryptoLib.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 解密
	decrypted, err := cryptoLib.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	// 验证解密结果
	if string(decrypted) != string(plaintext) {
		t.Errorf("解密结果不匹配，期望: %s, 实际: %s", plaintext, decrypted)
	}
}

// 测试未启用加密时直接返回原文
func TestCryptoLib_Disabled(t *testing.T) {
	config := &libs.CryptoConfig{
		Enabled:   false,
		Key:       "",
		Algorithm: "aes256gcm",
	}

	cryptoLib, err := libs.NewCryptoLib(config)
	if err != nil {
		t.Fatalf("创建CryptoLib失败: %v", err)
	}

	if cryptoLib.IsEnabled() {
		t.Error("未启用加密时，IsEnabled应该返回false")
	}

	plaintext := []byte("测试未启用加密")

	// 加密应该直接返回原文
	encrypted, err := cryptoLib.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}
	if string(encrypted) != string(plaintext) {
		t.Errorf("未启用加密时，加密应该返回原文")
	}

	// 解密应该直接返回原文
	decrypted, err := cryptoLib.Decrypt(plaintext)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}
	if string(decrypted) != string(plaintext) {
		t.Errorf("未启用加密时，解密应该返回原文")
	}
}

// 测试密钥长度不正确
func TestCryptoLib_InvalidKeyLength(t *testing.T) {
	// 测试AES-256-GCM密钥长度不足
	shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
	config := &libs.CryptoConfig{
		Enabled:   true,
		Key:       shortKey,
		Algorithm: "aes256gcm",
	}

	_, err := libs.NewCryptoLib(config)
	if err == nil {
		t.Error("密钥长度不足应该返回错误")
	}

	// 测试ChaCha20-Poly1305密钥长度不足
	config.Algorithm = "chacha20poly1305"
	_, err = libs.NewCryptoLib(config)
	if err == nil {
		t.Error("密钥长度不足应该返回错误")
	}
}

// 测试不支持的算法
func TestCryptoLib_UnsupportedAlgorithm(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	config := &libs.CryptoConfig{
		Enabled:   true,
		Key:       key,
		Algorithm: "invalid_algorithm",
	}

	_, err := libs.NewCryptoLib(config)
	if err == nil {
		t.Error("不支持的算法应该返回错误")
	}
}

// 测试Base64解码失败
func TestCryptoLib_InvalidBase64Key(t *testing.T) {
	config := &libs.CryptoConfig{
		Enabled:   true,
		Key:       "not-valid-base64!!!",
		Algorithm: "aes256gcm",
	}

	_, err := libs.NewCryptoLib(config)
	if err == nil {
		t.Error("无效的Base64密钥应该返回错误")
	}
}

// 测试空密钥时自动生成（仅用于测试）
func TestCryptoLib_EmptyKey(t *testing.T) {
	config := &libs.CryptoConfig{
		Enabled:   true,
		Key:       "",
		Algorithm: "aes256gcm",
	}

	cryptoLib, err := libs.NewCryptoLib(config)
	if err != nil {
		t.Fatalf("空密钥应该自动生成，不应返回错误: %v", err)
	}

	// 验证可以正常加解密
	plaintext := []byte("测试空密钥自动生成")
	ciphertext, err := cryptoLib.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	decrypted, err := cryptoLib.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("解密结果不匹配")
	}
}

// 测试多次加密产生不同密文（验证nonce随机性）
func TestCryptoLib_DifferentCiphertextEachTime(t *testing.T) {
	key := make([]byte, 32)
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	config := &libs.CryptoConfig{
		Enabled:   true,
		Key:       keyBase64,
		Algorithm: "aes256gcm",
	}

	cryptoLib, err := libs.NewCryptoLib(config)
	if err != nil {
		t.Fatalf("创建CryptoLib失败: %v", err)
	}

	plaintext := []byte("相同的明文")

	// 加密两次
	ciphertext1, err := cryptoLib.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("第一次加密失败: %v", err)
	}

	ciphertext2, err := cryptoLib.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("第二次加密失败: %v", err)
	}

	// 两次加密的密文应该不同（因为nonce不同）
	if string(ciphertext1) == string(ciphertext2) {
		t.Error("相同明文的两次加密应该产生不同的密文")
	}

	// 但两次解密都应该得到相同的明文
	decrypted1, _ := cryptoLib.Decrypt(ciphertext1)
	decrypted2, _ := cryptoLib.Decrypt(ciphertext2)

	if string(decrypted1) != string(plaintext) || string(decrypted2) != string(plaintext) {
		t.Error("解密结果应该与原文一致")
	}
}

// 测试空数据加密
func TestCryptoLib_EmptyData(t *testing.T) {
	key := make([]byte, 32)
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	config := &libs.CryptoConfig{
		Enabled:   true,
		Key:       keyBase64,
		Algorithm: "aes256gcm",
	}

	cryptoLib, err := libs.NewCryptoLib(config)
	if err != nil {
		t.Fatalf("创建CryptoLib失败: %v", err)
	}

	// 加密空数据
	plaintext := []byte{}
	ciphertext, err := cryptoLib.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("加密空数据失败: %v", err)
	}

	// 解密
	decrypted, err := cryptoLib.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("解密失败: %v", err)
	}

	if len(decrypted) != 0 {
		t.Errorf("解密空数据应该返回空数据")
	}
}

// 测试大数据加密
func TestCryptoLib_LargeData(t *testing.T) {
	key := make([]byte, 32)
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	config := &libs.CryptoConfig{
		Enabled:   true,
		Key:       keyBase64,
		Algorithm: "aes256gcm",
	}

	cryptoLib, err := libs.NewCryptoLib(config)
	if err != nil {
		t.Fatalf("创建CryptoLib失败: %v", err)
	}

	// 生成1MB的测试数据
	plaintext := make([]byte, 1024*1024)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}

	// 加密
	ciphertext, err := cryptoLib.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("加密大数据失败: %v", err)
	}

	// 解密
	decrypted, err := cryptoLib.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("解密大数据失败: %v", err)
	}

	// 验证数据
	if len(decrypted) != len(plaintext) {
		t.Errorf("解密数据长度不匹配，期望: %d, 实际: %d", len(plaintext), len(decrypted))
	}

	for i := range plaintext {
		if decrypted[i] != plaintext[i] {
			t.Errorf("解密数据在位置 %d 不匹配", i)
			break
		}
	}
}

// 测试解密篡改的密文
func TestCryptoLib_DecryptTamperedCiphertext(t *testing.T) {
	key := make([]byte, 32)
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	config := &libs.CryptoConfig{
		Enabled:   true,
		Key:       keyBase64,
		Algorithm: "aes256gcm",
	}

	cryptoLib, err := libs.NewCryptoLib(config)
	if err != nil {
		t.Fatalf("创建CryptoLib失败: %v", err)
	}

	plaintext := []byte("原始数据")
	ciphertext, err := cryptoLib.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("加密失败: %v", err)
	}

	// 篡改密文
	if len(ciphertext) > 20 {
		ciphertext[15] ^= 0xFF // 修改某个字节
	}

	// 解密应该失败
	_, err = cryptoLib.Decrypt(ciphertext)
	if err == nil {
		t.Error("解密篡改的密文应该失败")
	}
}

// 测试解密过短的密文
func TestCryptoLib_DecryptTooShortCiphertext(t *testing.T) {
	key := make([]byte, 32)
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	config := &libs.CryptoConfig{
		Enabled:   true,
		Key:       keyBase64,
		Algorithm: "aes256gcm",
	}

	cryptoLib, err := libs.NewCryptoLib(config)
	if err != nil {
		t.Fatalf("创建CryptoLib失败: %v", err)
	}

	// 解密过短的密文（少于nonce长度）
	shortCiphertext := []byte("short")
	_, err = cryptoLib.Decrypt(shortCiphertext)
	if err == nil {
		t.Error("解密过短的密文应该失败")
	}
}
