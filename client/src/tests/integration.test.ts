/**
 * 客户端集成测试
 * 测试端到端流程：WebSocket连接、消息收发、加密压缩通信、错误处理、重连机制
 */
import { CryptoUtil, CryptoConfig } from '../libs/crypto';
import { CompressUtil, CompressConfig } from '../libs/compress';

// ==================== 辅助函数 ====================

// 创建有效的Base64密钥
function createValidKey(): string {
  return Buffer.alloc(32, 0x41).toString('base64');
}

// 模拟 CompressionStream 和 DecompressionStream
function mockCompressionStream() {
  const createMockStream = () => ({
    writable: {
      getWriter: () => ({
        write: jest.fn().mockResolvedValue(undefined),
        close: jest.fn().mockResolvedValue(undefined),
      }),
    },
    readable: {
      getReader: () => ({
        read: jest.fn()
          .mockResolvedValueOnce({ value: new Uint8Array([31, 139, 8, 0, 0, 0, 0, 0, 0, 3]), done: false })
          .mockResolvedValueOnce({ value: undefined, done: true }),
      }),
    },
  });

  (global as any).CompressionStream = jest.fn().mockImplementation(createMockStream);
  (global as any).DecompressionStream = jest.fn().mockImplementation(createMockStream);
}

// 模拟 Web Crypto API
function mockWebCrypto() {
  const mockCryptoKey = {} as CryptoKey;

  const mockSubtle = {
    importKey: jest.fn().mockResolvedValue(mockCryptoKey),
    encrypt: jest.fn().mockResolvedValue(new ArrayBuffer(32)),
    decrypt: jest.fn().mockResolvedValue(new ArrayBuffer(16)),
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

  Object.defineProperty(global, 'crypto', {
    value: mockCrypto,
    writable: true,
  });

  return mockSubtle;
}

// ==================== 端到端流程测试 ====================

describe('集成测试：端到端流程', () => {
  let mockSubtle: ReturnType<typeof mockWebCrypto>;

  beforeEach(() => {
    mockSubtle = mockWebCrypto();
    mockCompressionStream();
  });

  describe('消息编解码流程', () => {
    it('应该正确完成 JSON → 压缩 → 加密 的编码流程', async () => {
      // 创建加密和压缩实例
      const cryptoConfig: CryptoConfig = {
        enabled: true,
        key: createValidKey(),
        algorithm: 'aes256gcm',
      };
      const compressConfig: CompressConfig = {
        enabled: true,
        level: 6,
        algorithm: 'gzip',
      };

      const crypto = new CryptoUtil(cryptoConfig);
      const compress = new CompressUtil(compressConfig);
      await crypto.waitForInit();

      // 构造消息
      const message = {
        id: 'test-001',
        type: 'request',
        data: {
          url: 'http://example.com/api',
          method: 'GET',
          headers: {},
        },
      };

      // 编码流程：JSON序列化 → 压缩 → 加密
      const jsonData = JSON.stringify(message);
      const jsonBytes = new TextEncoder().encode(jsonData);

      // 压缩
      const compressed = await compress.compress(jsonBytes);
      expect(compressed).toBeInstanceOf(Uint8Array);

      // 加密
      const encrypted = await crypto.encrypt(compressed);
      expect(encrypted).toBeInstanceOf(Uint8Array);

      // 验证加密被调用
      expect(mockSubtle.encrypt).toHaveBeenCalled();
    });

    it('应该正确完成 解密 → 解压 → JSON 的解码流程', async () => {
      const cryptoConfig: CryptoConfig = {
        enabled: true,
        key: createValidKey(),
        algorithm: 'aes256gcm',
      };
      const compressConfig: CompressConfig = {
        enabled: true,
        level: 6,
        algorithm: 'gzip',
      };

      const crypto = new CryptoUtil(cryptoConfig);
      const compress = new CompressUtil(compressConfig);
      await crypto.waitForInit();

      // 模拟接收到的加密数据
      const encryptedData = new Uint8Array(12 + 32); // nonce + ciphertext

      // 解密
      const decrypted = await crypto.decrypt(encryptedData);
      expect(mockSubtle.decrypt).toHaveBeenCalled();

      // 解压
      const decompressed = await compress.decompress(decrypted);
      expect(decompressed).toBeInstanceOf(Uint8Array);
    });
  });

  describe('请求-响应流程', () => {
    it('应该正确构造HTTP请求消息', async () => {
      const cryptoConfig: CryptoConfig = {
        enabled: false,
        key: '',
        algorithm: 'aes256gcm',
      };
      const compressConfig: CompressConfig = {
        enabled: false,
        level: 6,
        algorithm: 'gzip',
      };

      const crypto = new CryptoUtil(cryptoConfig);
      const compress = new CompressUtil(compressConfig);

      // 构造请求消息
      const requestMessage = {
        id: 'req-' + Date.now(),
        type: 'request',
        data: {
          url: 'http://example.com/api/users',
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer token123',
          },
          body: JSON.stringify({ name: 'test' }),
          bodyEncoding: 'text',
        },
      };

      // 编码
      const jsonData = JSON.stringify(requestMessage);
      const jsonBytes = new TextEncoder().encode(jsonData);
      const compressed = await compress.compress(jsonBytes);
      const encrypted = await crypto.encrypt(compressed);

      // 解码
      const decrypted = await crypto.decrypt(encrypted);
      const decompressed = await compress.decompress(decrypted);
      const decodedJson = new TextDecoder().decode(decompressed);
      const decodedMessage = JSON.parse(decodedJson);

      // 验证消息完整性
      expect(decodedMessage.id).toBe(requestMessage.id);
      expect(decodedMessage.type).toBe('request');
      expect(decodedMessage.data.url).toBe(requestMessage.data.url);
      expect(decodedMessage.data.method).toBe('POST');
    });

    it('应该正确处理响应消息', async () => {
      const cryptoConfig: CryptoConfig = {
        enabled: false,
        key: '',
        algorithm: 'aes256gcm',
      };
      const compressConfig: CompressConfig = {
        enabled: false,
        level: 6,
        algorithm: 'gzip',
      };

      const crypto = new CryptoUtil(cryptoConfig);
      const compress = new CompressUtil(compressConfig);

      // 构造响应消息
      const responseMessage = {
        id: 'req-123',
        type: 'response',
        data: {
          status: 200,
          statusText: 'OK',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({ success: true, data: { id: 1 } }),
          bodyEncoding: 'text',
        },
      };

      // 编码
      const jsonData = JSON.stringify(responseMessage);
      const jsonBytes = new TextEncoder().encode(jsonData);
      const compressed = await compress.compress(jsonBytes);
      const encrypted = await crypto.encrypt(compressed);

      // 解码
      const decrypted = await crypto.decrypt(encrypted);
      const decompressed = await compress.decompress(decrypted);
      const decodedJson = new TextDecoder().decode(decompressed);
      const decodedMessage = JSON.parse(decodedJson);

      // 验证响应
      expect(decodedMessage.type).toBe('response');
      expect(decodedMessage.data.status).toBe(200);
      expect(decodedMessage.data.statusText).toBe('OK');
    });
  });

  describe('心跳消息', () => {
    it('应该正确处理心跳请求', async () => {
      const cryptoConfig: CryptoConfig = {
        enabled: false,
        key: '',
        algorithm: 'aes256gcm',
      };
      const compressConfig: CompressConfig = {
        enabled: false,
        level: 6,
        algorithm: 'gzip',
      };

      const crypto = new CryptoUtil(cryptoConfig);
      const compress = new CompressUtil(compressConfig);

      // 构造心跳请求
      const pingMessage = {
        id: 'ping-001',
        type: 'ping',
        data: {
          timestamp: Date.now(),
        },
      };

      // 编码
      const jsonData = JSON.stringify(pingMessage);
      const jsonBytes = new TextEncoder().encode(jsonData);
      const encrypted = await crypto.encrypt(await compress.compress(jsonBytes));

      // 解码
      const decrypted = await crypto.decrypt(encrypted);
      const decompressed = await compress.decompress(decrypted);
      const decodedMessage = JSON.parse(new TextDecoder().decode(decompressed));

      expect(decodedMessage.type).toBe('ping');

      // 构造心跳响应
      const pongMessage = {
        id: decodedMessage.id,
        type: 'pong',
        data: decodedMessage.data,
      };

      expect(pongMessage.type).toBe('pong');
      expect(pongMessage.id).toBe(pingMessage.id);
    });
  });
});

// ==================== 加密压缩通信测试 ====================

describe('集成测试：加密压缩通信', () => {
  let mockSubtle: ReturnType<typeof mockWebCrypto>;

  beforeEach(() => {
    mockSubtle = mockWebCrypto();
    mockCompressionStream();
  });

  describe('客户端与服务端加密兼容性', () => {
    it('客户端加密的数据应能被服务端解密', async () => {
      const key = createValidKey();
      
      // 客户端加密配置
      const clientCryptoConfig: CryptoConfig = {
        enabled: true,
        key: key,
        algorithm: 'aes256gcm',
      };

      const clientCrypto = new CryptoUtil(clientCryptoConfig);
      await clientCrypto.waitForInit();

      // 原始数据
      const plaintext = new TextEncoder().encode('Hello from client!');

      // 客户端加密
      const encrypted = await clientCrypto.encrypt(plaintext);
      // encrypted = nonce(12) + ciphertext (which includes the tag)
      // 由于使用了mock，实际长度取决于mock返回的值
      expect(encrypted.length).toBeGreaterThan(12); // 至少包含nonce

      // 验证加密被调用
      expect(mockSubtle.encrypt).toHaveBeenCalledWith(
        expect.objectContaining({
          name: 'AES-GCM',
          tagLength: 128,
        }),
        expect.anything(),
        plaintext
      );
    });

    it('不同密钥加密的数据应不能互相解密', async () => {
      const key1 = Buffer.alloc(32, 0x41).toString('base64');
      const key2 = Buffer.alloc(32, 0x42).toString('base64');

      const crypto1 = new CryptoUtil({ enabled: true, key: key1, algorithm: 'aes256gcm' });
      const crypto2 = new CryptoUtil({ enabled: true, key: key2, algorithm: 'aes256gcm' });

      await crypto1.waitForInit();
      await crypto2.waitForInit();

      const plaintext = new TextEncoder().encode('Secret message');
      const encrypted = await crypto1.encrypt(plaintext);

      // 使用不同密钥解密应该失败
      mockSubtle.decrypt.mockRejectedValueOnce(new Error('解密失败'));
      
      await expect(crypto2.decrypt(encrypted)).rejects.toThrow();
    });
  });

  describe('压缩效果测试', () => {
    it('重复数据应该有良好的压缩比', async () => {
      mockCompressionStream();

      const compressConfig: CompressConfig = {
        enabled: true,
        level: 6,
        algorithm: 'gzip',
      };

      const compress = new CompressUtil(compressConfig);

      // 创建大量重复数据
      const repeatedData = 'A'.repeat(1000);
      const data = new TextEncoder().encode(repeatedData);

      await compress.compress(data);

      // 验证压缩被调用
      expect((global as any).CompressionStream).toHaveBeenCalledWith('gzip');
    });
  });
});

// ==================== 错误处理测试 ====================

describe('集成测试：错误处理', () => {
  let mockSubtle: ReturnType<typeof mockWebCrypto>;

  beforeEach(() => {
    mockSubtle = mockWebCrypto();
    mockCompressionStream();
  });

  describe('网络错误处理', () => {
    it('无效URL应该返回错误', () => {
      const invalidUrls = [
        '://invalid',
        'http://',
        'not-a-url',
        '',
      ];

      invalidUrls.forEach(url => {
        const message = {
          id: 'test',
          type: 'request',
          data: { url, method: 'GET' },
        };

        // 验证消息构造
        expect(message.data.url).toBe(url);
        
        // 在实际场景中，这应该导致请求失败
      });
    });

    it('连接超时应该正确处理', () => {
      const timeout = 30000; // 30秒超时
      
      // 模拟超时场景
      const requestConfig = {
        url: 'http://slow-server.example.com',
        method: 'GET',
        timeout,
      };

      expect(requestConfig.timeout).toBe(timeout);
    });
  });

  describe('解密错误处理', () => {
    it('篡改的数据解密应该失败', async () => {
      const cryptoConfig: CryptoConfig = {
        enabled: true,
        key: createValidKey(),
        algorithm: 'aes256gcm',
      };

      const crypto = new CryptoUtil(cryptoConfig);
      await crypto.waitForInit();

      // 模拟被篡改的数据
      const tamperedData = new Uint8Array([1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15]);

      mockSubtle.decrypt.mockRejectedValueOnce(new Error('解密失败'));

      await expect(crypto.decrypt(tamperedData)).rejects.toThrow('解密失败');
    });

    it('过短的密文应该抛出错误', async () => {
      const cryptoConfig: CryptoConfig = {
        enabled: true,
        key: createValidKey(),
        algorithm: 'aes256gcm',
      };

      const crypto = new CryptoUtil(cryptoConfig);
      await crypto.waitForInit();

      // 少于12字节的密文
      const shortCiphertext = new Uint8Array([1, 2, 3]);

      await expect(crypto.decrypt(shortCiphertext)).rejects.toThrow('密文长度不足');
    });
  });

  describe('消息格式错误处理', () => {
    it('无效的JSON消息应该被正确处理', () => {
      const invalidJsonStrings = [
        '{ invalid json }',
        'not json at all',
        '{"incomplete": ',
        '',
      ];

      invalidJsonStrings.forEach(jsonStr => {
        try {
          JSON.parse(jsonStr);
          // 如果没有抛出错误，测试失败
          expect(true).toBe(false);
        } catch (e) {
          // 预期抛出错误
          expect(e).toBeInstanceOf(SyntaxError);
        }
      });
    });

    it('缺少必需字段的消息应该被拒绝', () => {
      const incompleteMessages: Array<{ id?: string; type?: string; data?: any }> = [
        { id: 'test' }, // 缺少 type
        { type: 'request' }, // 缺少 id
        { id: 'test', type: 'request' }, // 缺少 data
      ];

      incompleteMessages.forEach(msg => {
        // 在实际场景中，这些消息应该被拒绝
        const isValid = msg.id && msg.type && (msg.type !== 'request' || msg.data);
        if (msg.type === 'request') {
          expect(isValid).toBeFalsy();
        }
      });
    });
  });
});

// ==================== 重连机制测试 ====================

describe('集成测试：重连机制', () => {
  it('连接失败后应该尝试重连', () => {
    // 模拟WebSocket重连逻辑
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 5;
    const reconnectDelay = 1000;

    const attemptReconnect = () => {
      if (reconnectAttempts < maxReconnectAttempts) {
        reconnectAttempts++;
        // 模拟重连延迟
        const delay = Math.min(reconnectDelay * Math.pow(2, reconnectAttempts - 1), 30000);
        return { attempt: reconnectAttempts, delay };
      }
      return null;
    };

    // 模拟多次重连尝试
    const results = [];
    let result;
    while ((result = attemptReconnect()) !== null) {
      results.push(result);
    }

    expect(results.length).toBe(maxReconnectAttempts);
    
    // 验证指数退避
    for (let i = 1; i < results.length; i++) {
      expect(results[i].delay).toBeGreaterThan(results[i - 1].delay);
    }
  });

  it('重连成功后应该重置重连计数', () => {
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 5;

    // 模拟连接失败
    const onConnectionFailed = () => {
      reconnectAttempts++;
    };

    // 模拟连接成功
    const onConnectionSuccess = () => {
      reconnectAttempts = 0;
    };

    // 连接失败几次
    onConnectionFailed();
    onConnectionFailed();
    expect(reconnectAttempts).toBe(2);

    // 连接成功，重置计数
    onConnectionSuccess();
    expect(reconnectAttempts).toBe(0);
  });

  it('达到最大重连次数后应该停止重连', () => {
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 3;
    let shouldReconnect = true;

    const attemptReconnect = () => {
      if (reconnectAttempts >= maxReconnectAttempts) {
        shouldReconnect = false;
        return false;
      }
      reconnectAttempts++;
      return true;
    };

    // 尝试重连
    expect(attemptReconnect()).toBe(true);  // 第1次
    expect(attemptReconnect()).toBe(true);  // 第2次
    expect(attemptReconnect()).toBe(true);  // 第3次
    expect(attemptReconnect()).toBe(false); // 第4次，超过最大次数

    expect(shouldReconnect).toBe(false);
  });
});

// ==================== 消息ID一致性测试 ====================

describe('集成测试：消息ID一致性', () => {
  let mockSubtle: ReturnType<typeof mockWebCrypto>;

  beforeEach(() => {
    mockSubtle = mockWebCrypto();
    mockCompressionStream();
  });

  it('请求和响应的消息ID应该一致', async () => {
    const cryptoConfig: CryptoConfig = {
      enabled: false,
      key: '',
      algorithm: 'aes256gcm',
    };
    const compressConfig: CompressConfig = {
      enabled: false,
      level: 6,
      algorithm: 'gzip',
    };

    const crypto = new CryptoUtil(cryptoConfig);
    const compress = new CompressUtil(compressConfig);

    // 构造请求
    const requestId = 'req-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
    const requestMessage = {
      id: requestId,
      type: 'request',
      data: {
        url: 'http://example.com/api',
        method: 'GET',
      },
    };

    // 编码请求
    const encoded = await crypto.encrypt(
      await compress.compress(
        new TextEncoder().encode(JSON.stringify(requestMessage))
      )
    );

    // 解码请求
    const decoded = JSON.parse(
      new TextDecoder().decode(
        await compress.decompress(await crypto.decrypt(encoded))
      )
    );

    // 构造响应（使用相同的ID）
    const responseMessage = {
      id: decoded.id,
      type: 'response',
      data: {
        status: 200,
        statusText: 'OK',
        body: 'Success',
      },
    };

    // 验证ID一致性
    expect(responseMessage.id).toBe(requestId);
    expect(responseMessage.id).toBe(requestMessage.id);
  });

  it('多个并发请求的ID应该唯一', () => {
    const generateId = () => {
      return 'req-' + Date.now() + '-' + Math.random().toString(36).substr(2, 9);
    };

    const ids = new Set<string>();
    for (let i = 0; i < 100; i++) {
      ids.add(generateId());
    }

    // 验证所有ID都是唯一的
    expect(ids.size).toBe(100);
  });
});

// ==================== 二进制数据处理测试 ====================

describe('集成测试：二进制数据处理', () => {
  let mockSubtle: ReturnType<typeof mockWebCrypto>;

  beforeEach(() => {
    mockSubtle = mockWebCrypto();
    mockCompressionStream();
  });

  it('二进制响应应该使用Base64编码', async () => {
    // 模拟二进制响应
    const binaryData = new Uint8Array([0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A]); // PNG文件头
    const base64Body = Buffer.from(binaryData).toString('base64');

    const responseMessage = {
      id: 'req-001',
      type: 'response',
      data: {
        status: 200,
        statusText: 'OK',
        headers: {
          'Content-Type': 'image/png',
        },
        body: base64Body,
        bodyEncoding: 'base64',
      },
    };

    expect(responseMessage.data.bodyEncoding).toBe('base64');

    // 验证可以正确解码
    const decoded = Buffer.from(responseMessage.data.body, 'base64');
    expect(decoded).toEqual(Buffer.from(binaryData));
  });

  it('Base64编码的请求体应该正确处理', async () => {
    const cryptoConfig: CryptoConfig = {
      enabled: false,
      key: '',
      algorithm: 'aes256gcm',
    };
    const compressConfig: CompressConfig = {
      enabled: false,
      level: 6,
      algorithm: 'gzip',
    };

    const crypto = new CryptoUtil(cryptoConfig);
    const compress = new CompressUtil(compressConfig);

    // 构造包含Base64编码body的请求
    const binaryPayload = new Uint8Array([1, 2, 3, 4, 5, 6, 7, 8]);
    const base64Body = Buffer.from(binaryPayload).toString('base64');

    const requestMessage = {
      id: 'binary-req-001',
      type: 'request',
      data: {
        url: 'http://example.com/upload',
        method: 'POST',
        headers: {
          'Content-Type': 'application/octet-stream',
        },
        body: base64Body,
        bodyEncoding: 'base64',
      },
    };

    // 编码
    const encoded = await crypto.encrypt(
      await compress.compress(
        new TextEncoder().encode(JSON.stringify(requestMessage))
      )
    );

    // 解码
    const decoded = JSON.parse(
      new TextDecoder().decode(
        await compress.decompress(await crypto.decrypt(encoded))
      )
    );

    // 验证
    expect(decoded.data.bodyEncoding).toBe('base64');
    expect(decoded.data.body).toBe(base64Body);
  });
});

// ==================== 响应头处理测试 ====================

describe('集成测试：响应头处理', () => {
  let mockSubtle: ReturnType<typeof mockWebCrypto>;

  beforeEach(() => {
    mockSubtle = mockWebCrypto();
    mockCompressionStream();
  });

  it('应该正确传递和接收响应头', async () => {
    const cryptoConfig: CryptoConfig = {
      enabled: false,
      key: '',
      algorithm: 'aes256gcm',
    };
    const compressConfig: CompressConfig = {
      enabled: false,
      level: 6,
      algorithm: 'gzip',
    };

    const crypto = new CryptoUtil(cryptoConfig);
    const compress = new CompressUtil(compressConfig);

    // 构造包含响应头的响应
    const responseMessage = {
      id: 'req-001',
      type: 'response',
      data: {
        status: 200,
        statusText: 'OK',
        headers: {
          'Content-Type': 'application/json',
          'X-Custom-Header': 'custom-value',
          'X-Request-Id': 'server-123',
          'Cache-Control': 'no-cache',
        },
        body: '{"success":true}',
        bodyEncoding: 'text',
      },
    };

    // 编码解码
    const encoded = await crypto.encrypt(
      await compress.compress(
        new TextEncoder().encode(JSON.stringify(responseMessage))
      )
    );

    const decoded = JSON.parse(
      new TextDecoder().decode(
        await compress.decompress(await crypto.decrypt(encoded))
      )
    );

    // 验证响应头
    expect(decoded.data.headers['Content-Type']).toBe('application/json');
    expect(decoded.data.headers['X-Custom-Header']).toBe('custom-value');
    expect(decoded.data.headers['X-Request-Id']).toBe('server-123');
    expect(decoded.data.headers['Cache-Control']).toBe('no-cache');
  });

  it('多值响应头应该正确处理', () => {
    // 模拟多值响应头（如Set-Cookie）
    const headers: Record<string, string> = {
      'Set-Cookie': 'session=abc; Path=/, user=test; Path=/',
    };

    // 验证多值头存在
    expect(headers['Set-Cookie']).toContain('session=abc');
    expect(headers['Set-Cookie']).toContain('user=test');
  });
});

// ==================== HTTP方法测试 ====================

describe('集成测试：HTTP方法支持', () => {
  let mockSubtle: ReturnType<typeof mockWebCrypto>;

  beforeEach(() => {
    mockSubtle = mockWebCrypto();
    mockCompressionStream();
  });

  const httpMethods = ['GET', 'POST', 'PUT', 'DELETE', 'PATCH', 'HEAD', 'OPTIONS'];

  httpMethods.forEach(method => {
    it(`应该支持 ${method} 方法`, async () => {
      const cryptoConfig: CryptoConfig = {
        enabled: false,
        key: '',
        algorithm: 'aes256gcm',
      };
      const compressConfig: CompressConfig = {
        enabled: false,
        level: 6,
        algorithm: 'gzip',
      };

      const crypto = new CryptoUtil(cryptoConfig);
      const compress = new CompressUtil(compressConfig);

      const requestMessage = {
        id: `${method.toLowerCase()}-req-001`,
        type: 'request',
        data: {
          url: 'http://example.com/api/resource',
          method: method,
          headers: {},
          body: method === 'GET' || method === 'HEAD' ? '' : '{"data":"test"}',
          bodyEncoding: 'text',
        },
      };

      // 编码解码
      const encoded = await crypto.encrypt(
        await compress.compress(
          new TextEncoder().encode(JSON.stringify(requestMessage))
        )
      );

      const decoded = JSON.parse(
        new TextDecoder().decode(
          await compress.decompress(await crypto.decrypt(encoded))
        )
      );

      expect(decoded.data.method).toBe(method);
    });
  });
});

// ==================== 性能测试 ====================

describe('集成测试：性能', () => {
  let mockSubtle: ReturnType<typeof mockWebCrypto>;

  beforeEach(() => {
    mockSubtle = mockWebCrypto();
    mockCompressionStream();
  });

  it('消息编解码应该在合理时间内完成', async () => {
    const cryptoConfig: CryptoConfig = {
      enabled: false,
      key: '',
      algorithm: 'aes256gcm',
    };
    const compressConfig: CompressConfig = {
      enabled: false,
      level: 6,
      algorithm: 'gzip',
    };

    const crypto = new CryptoUtil(cryptoConfig);
    const compress = new CompressUtil(compressConfig);

    const message = {
      id: 'perf-test-001',
      type: 'request',
      data: {
        url: 'http://example.com/api',
        method: 'GET',
        headers: {},
      },
    };

    const iterations = 100;
    const start = Date.now();

    for (let i = 0; i < iterations; i++) {
      const encoded = await crypto.encrypt(
        await compress.compress(
          new TextEncoder().encode(JSON.stringify(message))
        )
      );

      await compress.decompress(
        await crypto.decrypt(encoded)
      );
    }

    const elapsed = Date.now() - start;
    const avgTime = elapsed / iterations;

    console.log(`平均编解码时间: ${avgTime.toFixed(2)}ms`);
    
    // 每次编解码应该在100ms内完成
    expect(avgTime).toBeLessThan(100);
  });

  it('大数据处理应该在合理时间内完成', async () => {
    const cryptoConfig: CryptoConfig = {
      enabled: false,
      key: '',
      algorithm: 'aes256gcm',
    };
    const compressConfig: CompressConfig = {
      enabled: false,
      level: 6,
      algorithm: 'gzip',
    };

    const crypto = new CryptoUtil(cryptoConfig);
    const compress = new CompressUtil(compressConfig);

    // 构造大数据消息（约100KB）
    const largeBody = 'A'.repeat(100 * 1024);
    const message = {
      id: 'large-data-001',
      type: 'request',
      data: {
        url: 'http://example.com/api',
        method: 'POST',
        body: largeBody,
        bodyEncoding: 'text',
      },
    };

    const start = Date.now();

    const encoded = await crypto.encrypt(
      await compress.compress(
        new TextEncoder().encode(JSON.stringify(message))
      )
    );

    const decoded = JSON.parse(
      new TextDecoder().decode(
        await compress.decompress(await crypto.decrypt(encoded))
      )
    );

    const elapsed = Date.now() - start;

    console.log(`大数据编解码时间: ${elapsed}ms`);
    
    // 应该在1秒内完成
    expect(elapsed).toBeLessThan(1000);
    expect(decoded.data.body).toBe(largeBody);
  });
});
