/**
 * 加密工具库测试
 */
import { CryptoUtil, CryptoConfig } from '../libs/crypto';

// 模拟 Web Crypto API
const mockCryptoKey = {} as CryptoKey;

const mockSubtle = {
  importKey: jest.fn().mockResolvedValue(mockCryptoKey),
  encrypt: jest.fn(),
  decrypt: jest.fn(),
};

const mockCrypto = {
  subtle: mockSubtle,
  getRandomValues: jest.fn((array: Uint8Array) => {
    for (let i = 0; i < array.length; i++) {
      array[i] = Math.floor(Math.random() * 256);
    }
    return array;
  }),
};

// 替换全局 crypto
Object.defineProperty(global, 'crypto', {
  value: mockCrypto,
  writable: true,
});

describe('CryptoUtil', () => {
  // 生成有效的32字节Base64密钥
  const validKey = Buffer.alloc(32, 0x41).toString('base64');

  describe('构造函数和初始化', () => {
    it('未启用加密时应该正常创建实例', () => {
      const config: CryptoConfig = {
        enabled: false,
        key: '',
        algorithm: 'aes256gcm',
      };

      const crypto = new CryptoUtil(config);
      expect(crypto.isEnabled()).toBe(false);
    });

    it('启用加密时应该初始化密钥', async () => {
      const config: CryptoConfig = {
        enabled: true,
        key: validKey,
        algorithm: 'aes256gcm',
      };

      const crypto = new CryptoUtil(config);
      await crypto.waitForInit();
      expect(mockSubtle.importKey).toHaveBeenCalled();
    });

    it('密钥长度不正确时应该抛出错误', async () => {
      const invalidKey = Buffer.alloc(16, 0x41).toString('base64'); // 16字节，不是32字节
      const config: CryptoConfig = {
        enabled: true,
        key: invalidKey,
        algorithm: 'aes256gcm',
      };

      const crypto = new CryptoUtil(config);
      await expect(crypto.waitForInit()).rejects.toThrow('密钥长度必须为32字节');
    });

    it('ChaCha20-Poly1305应该不支持', async () => {
      const config: CryptoConfig = {
        enabled: true,
        key: validKey,
        algorithm: 'chacha20poly1305',
      };

      const crypto = new CryptoUtil(config);
      await expect(crypto.waitForInit()).rejects.toThrow('ChaCha20-Poly1305 在浏览器中不支持');
    });
  });

  describe('加密解密功能', () => {
    it('未启用加密时加密应该返回原文', async () => {
      const config: CryptoConfig = {
        enabled: false,
        key: '',
        algorithm: 'aes256gcm',
      };

      const crypto = new CryptoUtil(config);
      const plaintext = new Uint8Array([1, 2, 3, 4, 5]);

      const encrypted = await crypto.encrypt(plaintext);
      expect(encrypted).toEqual(plaintext);
    });

    it('未启用加密时解密应该返回原文', async () => {
      const config: CryptoConfig = {
        enabled: false,
        key: '',
        algorithm: 'aes256gcm',
      };

      const crypto = new CryptoUtil(config);
      const ciphertext = new Uint8Array([1, 2, 3, 4, 5]);

      const decrypted = await crypto.decrypt(ciphertext);
      expect(decrypted).toEqual(ciphertext);
    });

    it('AES-256-GCM加密应该正确调用Web Crypto API', async () => {
      const config: CryptoConfig = {
        enabled: true,
        key: validKey,
        algorithm: 'aes256gcm',
      };

      // 模拟加密结果
      const mockCiphertext = new Uint8Array(32);
      mockSubtle.encrypt.mockResolvedValueOnce(mockCiphertext.buffer);

      const crypto = new CryptoUtil(config);
      await crypto.waitForInit();

      const plaintext = new Uint8Array([72, 101, 108, 108, 111]); // "Hello"
      const encrypted = await crypto.encrypt(plaintext);

      // 验证加密被调用
      expect(mockSubtle.encrypt).toHaveBeenCalledWith(
        expect.objectContaining({
          name: 'AES-GCM',
          iv: expect.any(Uint8Array),
          tagLength: 128,
        }),
        mockCryptoKey,
        plaintext
      );

      // 验证结果包含nonce（12字节）和密文
      expect(encrypted.length).toBe(12 + 32);
    });

    it('AES-256-GCM解密应该正确调用Web Crypto API', async () => {
      const config: CryptoConfig = {
        enabled: true,
        key: validKey,
        algorithm: 'aes256gcm',
      };

      // 模拟解密结果
      const mockPlaintext = new Uint8Array([72, 101, 108, 108, 111]); // "Hello"
      mockSubtle.decrypt.mockResolvedValueOnce(mockPlaintext.buffer);

      const crypto = new CryptoUtil(config);
      await crypto.waitForInit();

      // 模拟密文（12字节nonce + 密文）
      const ciphertext = new Uint8Array(12 + 16);
      const decrypted = await crypto.decrypt(ciphertext);

      // 验证解密被调用
      expect(mockSubtle.decrypt).toHaveBeenCalledWith(
        expect.objectContaining({
          name: 'AES-GCM',
          iv: expect.any(Uint8Array),
          tagLength: 128,
        }),
        mockCryptoKey,
        expect.any(Uint8Array)
      );

      expect(decrypted).toEqual(mockPlaintext);
    });

    it('密文过短时解密应该抛出错误', async () => {
      const config: CryptoConfig = {
        enabled: true,
        key: validKey,
        algorithm: 'aes256gcm',
      };

      const crypto = new CryptoUtil(config);
      await crypto.waitForInit();

      // 太短的密文（少于12字节nonce）
      const shortCiphertext = new Uint8Array([1, 2, 3]);
      await expect(crypto.decrypt(shortCiphertext)).rejects.toThrow('密文长度不足');
    });

    it('解密失败时应该抛出错误', async () => {
      const config: CryptoConfig = {
        enabled: true,
        key: validKey,
        algorithm: 'aes256gcm',
      };

      mockSubtle.decrypt.mockRejectedValueOnce(new Error('解密失败'));

      const crypto = new CryptoUtil(config);
      await crypto.waitForInit();

      const ciphertext = new Uint8Array(12 + 16); // nonce + ciphertext
      await expect(crypto.decrypt(ciphertext)).rejects.toThrow('解密失败');
    });
  });

  describe('辅助方法', () => {
    it('isEnabled应该返回正确的状态', () => {
      const configEnabled: CryptoConfig = {
        enabled: true,
        key: validKey,
        algorithm: 'aes256gcm',
      };
      const cryptoEnabled = new CryptoUtil(configEnabled);
      expect(cryptoEnabled.isEnabled()).toBe(true);

      const configDisabled: CryptoConfig = {
        enabled: false,
        key: '',
        algorithm: 'aes256gcm',
      };
      const cryptoDisabled = new CryptoUtil(configDisabled);
      expect(cryptoDisabled.isEnabled()).toBe(false);
    });
  });
});
