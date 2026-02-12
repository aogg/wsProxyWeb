// 加密工具库

// 加密配置接口
export interface CryptoConfig {
  enabled: boolean;
  key: string; // Base64编码的32字节密钥
  algorithm: 'aes256gcm' | 'chacha20poly1305';
}

// 加密工具类
export class CryptoUtil {
  private enabled: boolean;
  private key: CryptoKey | null = null;
  private algorithm: string;
  private keyData: Uint8Array | null = null;

  constructor(config: CryptoConfig) {
    this.enabled = config.enabled;
    this.algorithm = config.algorithm;
    
    if (config.enabled) {
      this.initKey(config.key);
    }
  }

  /**
   * 初始化加密密钥
   */
  private async initKey(keyBase64: string): Promise<void> {
    try {
      // 解码Base64密钥
      const keyBytes = this.base64ToUint8Array(keyBase64);
      
      if (keyBytes.length !== 32) {
        throw new Error(`密钥长度必须为32字节，当前长度: ${keyBytes.length}`);
      }

      this.keyData = keyBytes;

      // 根据算法类型初始化密钥
      if (this.algorithm === 'aes256gcm') {
        // AES-256-GCM
        this.key = await crypto.subtle.importKey(
          'raw',
          keyBytes,
          { name: 'AES-GCM' },
          false,
          ['encrypt', 'decrypt']
        );
      } else if (this.algorithm === 'chacha20poly1305') {
        // ChaCha20-Poly1305 在 Web Crypto API 中不支持
        // 使用 AES-GCM 作为替代，或者抛出错误
        throw new Error('ChaCha20-Poly1305 在浏览器中不支持，请使用 aes256gcm');
      } else {
        throw new Error(`不支持的加密算法: ${this.algorithm}`);
      }
    } catch (error) {
      console.error('初始化加密密钥失败:', error);
      this.enabled = false;
      throw error;
    }
  }

  /**
   * 加密数据
   * 返回格式：nonce + ciphertext + tag（Uint8Array）
   */
  async encrypt(plaintext: Uint8Array): Promise<Uint8Array> {
    if (!this.enabled || !this.key) {
      return plaintext;
    }

    if (this.algorithm === 'aes256gcm') {
      return this.encryptAES256GCM(plaintext);
    } else {
      throw new Error(`不支持的加密算法: ${this.algorithm}`);
    }
  }

  /**
   * 解密数据
   * 输入格式：nonce + ciphertext + tag（Uint8Array）
   */
  async decrypt(ciphertext: Uint8Array): Promise<Uint8Array> {
    if (!this.enabled || !this.key) {
      return ciphertext;
    }

    if (this.algorithm === 'aes256gcm') {
      return this.decryptAES256GCM(ciphertext);
    } else {
      throw new Error(`不支持的加密算法: ${this.algorithm}`);
    }
  }

  /**
   * 使用AES-256-GCM加密
   */
  private async encryptAES256GCM(plaintext: Uint8Array): Promise<Uint8Array> {
    if (!this.key) {
      throw new Error('加密密钥未初始化');
    }

    // 生成12字节的nonce
    const nonce = crypto.getRandomValues(new Uint8Array(12));

    // 加密
    const ciphertext = await crypto.subtle.encrypt(
      {
        name: 'AES-GCM',
        iv: nonce,
        tagLength: 128 // 16字节的认证标签
      },
      this.key,
      plaintext
    );

    // 组合 nonce + ciphertext
    const result = new Uint8Array(nonce.length + ciphertext.byteLength);
    result.set(nonce, 0);
    result.set(new Uint8Array(ciphertext), nonce.length);

    return result;
  }

  /**
   * 使用AES-256-GCM解密
   */
  private async decryptAES256GCM(ciphertext: Uint8Array): Promise<Uint8Array> {
    if (!this.key) {
      throw new Error('加密密钥未初始化');
    }

    // 提取nonce（前12字节）
    const nonceSize = 12;
    if (ciphertext.length < nonceSize) {
      throw new Error('密文长度不足');
    }

    const nonce = ciphertext.slice(0, nonceSize);
    const encryptedData = ciphertext.slice(nonceSize);

    // 解密
    try {
      const plaintext = await crypto.subtle.decrypt(
        {
          name: 'AES-GCM',
          iv: nonce,
          tagLength: 128
        },
        this.key!,
        encryptedData
      );

      return new Uint8Array(plaintext);
    } catch (error) {
      throw new Error(`解密失败: ${error}`);
    }
  }

  /**
   * 检查加密是否启用
   */
  isEnabled(): boolean {
    return this.enabled;
  }

  /**
   * Base64字符串转Uint8Array
   */
  private base64ToUint8Array(base64: string): Uint8Array {
    // 移除可能的填充和空白字符
    const cleanBase64 = base64.replace(/\s/g, '');
    
    // 解码Base64
    const binaryString = atob(cleanBase64);
    const bytes = new Uint8Array(binaryString.length);
    
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i);
    }
    
    return bytes;
  }

  /**
   * Uint8Array转Base64字符串
   */
  private uint8ArrayToBase64(bytes: Uint8Array): string {
    let binary = '';
    for (let i = 0; i < bytes.length; i++) {
      binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary);
  }
}
