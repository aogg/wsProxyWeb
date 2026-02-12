// 加密工具库
package libs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/chacha20poly1305"
)

// CryptoLib 加密库结构
type CryptoLib struct {
	enabled   bool
	key       []byte
	algorithm string
}

// NewCryptoLib 创建新的加密库实例
func NewCryptoLib(config *CryptoConfig) (*CryptoLib, error) {
	lib := &CryptoLib{
		enabled:   config.Enabled,
		algorithm: config.Algorithm,
	}

	if !config.Enabled {
		return lib, nil
	}

	// 解析密钥
	var key []byte
	var err error
	if config.Key == "" {
		// 如果密钥为空，生成一个随机密钥（仅用于测试）
		key = make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("生成随机密钥失败: %v", err)
		}
	} else {
		// 从Base64解码密钥
		key, err = base64.StdEncoding.DecodeString(config.Key)
		if err != nil {
			return nil, fmt.Errorf("解码密钥失败: %v", err)
		}
	}

	// 验证密钥长度
	switch config.Algorithm {
	case "aes256gcm":
		if len(key) != 32 {
			return nil, fmt.Errorf("AES-256-GCM需要32字节密钥，当前长度: %d", len(key))
		}
	case "chacha20poly1305":
		if len(key) != 32 {
			return nil, fmt.Errorf("ChaCha20-Poly1305需要32字节密钥，当前长度: %d", len(key))
		}
	default:
		return nil, fmt.Errorf("不支持的加密算法: %s", config.Algorithm)
	}

	lib.key = key
	return lib, nil
}

// Encrypt 加密数据
// 返回格式：nonce + ciphertext + tag（Base64编码）
func (c *CryptoLib) Encrypt(plaintext []byte) ([]byte, error) {
	if !c.enabled {
		return plaintext, nil
	}

	switch c.algorithm {
	case "aes256gcm":
		return c.encryptAES256GCM(plaintext)
	case "chacha20poly1305":
		return c.encryptChaCha20Poly1305(plaintext)
	default:
		return nil, fmt.Errorf("不支持的加密算法: %s", c.algorithm)
	}
}

// Decrypt 解密数据
// 输入格式：nonce + ciphertext + tag（Base64编码）
func (c *CryptoLib) Decrypt(ciphertext []byte) ([]byte, error) {
	if !c.enabled {
		return ciphertext, nil
	}

	switch c.algorithm {
	case "aes256gcm":
		return c.decryptAES256GCM(ciphertext)
	case "chacha20poly1305":
		return c.decryptChaCha20Poly1305(ciphertext)
	default:
		return nil, fmt.Errorf("不支持的加密算法: %s", c.algorithm)
	}
}

// encryptAES256GCM 使用AES-256-GCM加密
func (c *CryptoLib) encryptAES256GCM(plaintext []byte) ([]byte, error) {
	// 创建AES cipher
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %v", err)
	}

	// 创建GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM失败: %v", err)
	}

	// 生成nonce（12字节）
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成nonce失败: %v", err)
	}

	// 加密
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// decryptAES256GCM 使用AES-256-GCM解密
func (c *CryptoLib) decryptAES256GCM(ciphertext []byte) ([]byte, error) {
	// 创建AES cipher
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, fmt.Errorf("创建AES cipher失败: %v", err)
	}

	// 创建GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("创建GCM失败: %v", err)
	}

	// 提取nonce
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("密文长度不足")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// 解密
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %v", err)
	}

	return plaintext, nil
}

// encryptChaCha20Poly1305 使用ChaCha20-Poly1305加密
func (c *CryptoLib) encryptChaCha20Poly1305(plaintext []byte) ([]byte, error) {
	// 创建ChaCha20-Poly1305 cipher
	aead, err := chacha20poly1305.New(c.key)
	if err != nil {
		return nil, fmt.Errorf("创建ChaCha20-Poly1305 cipher失败: %v", err)
	}

	// 生成nonce（12字节）
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("生成nonce失败: %v", err)
	}

	// 加密
	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// decryptChaCha20Poly1305 使用ChaCha20-Poly1305解密
func (c *CryptoLib) decryptChaCha20Poly1305(ciphertext []byte) ([]byte, error) {
	// 创建ChaCha20-Poly1305 cipher
	aead, err := chacha20poly1305.New(c.key)
	if err != nil {
		return nil, fmt.Errorf("创建ChaCha20-Poly1305 cipher失败: %v", err)
	}

	// 提取nonce
	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("密文长度不足")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// 解密
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %v", err)
	}

	return plaintext, nil
}

// IsEnabled 检查加密是否启用
func (c *CryptoLib) IsEnabled() bool {
	return c.enabled
}
