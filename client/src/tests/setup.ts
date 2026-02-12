// Jest 测试设置文件

// 模拟 Chrome API
const chromeMock = {
  runtime: {
    sendMessage: jest.fn(),
    onMessage: {
      addListener: jest.fn(),
      removeListener: jest.fn(),
    },
    getURL: jest.fn((path: string) => `chrome-extension://test-id/${path}`),
  },
  storage: {
    local: {
      get: jest.fn((keys, callback) => {
        if (callback) {
          callback({});
        }
        return Promise.resolve({});
      }),
      set: jest.fn((items, callback) => {
        if (callback) {
          callback();
        }
        return Promise.resolve();
      }),
      remove: jest.fn((keys, callback) => {
        if (callback) {
          callback();
        }
        return Promise.resolve();
      }),
    },
    sync: {
      get: jest.fn((keys, callback) => {
        if (callback) {
          callback({});
        }
        return Promise.resolve({});
      }),
      set: jest.fn((items, callback) => {
        if (callback) {
          callback();
        }
        return Promise.resolve();
      }),
    },
  },
  tabs: {
    query: jest.fn(),
    sendMessage: jest.fn(),
    create: jest.fn(),
  },
  webRequest: {
    onBeforeRequest: {
      addListener: jest.fn(),
      removeListener: jest.fn(),
    },
    onHeadersReceived: {
      addListener: jest.fn(),
      removeListener: jest.fn(),
    },
  },
};

// 设置全局 chrome 对象
(global as any).chrome = chromeMock;

// 模拟 Web Crypto API
const cryptoMock = {
  subtle: {
    importKey: jest.fn().mockResolvedValue({}),
    encrypt: jest.fn().mockResolvedValue(new ArrayBuffer(32)),
    decrypt: jest.fn().mockResolvedValue(new ArrayBuffer(32)),
    generateKey: jest.fn().mockResolvedValue({}),
  },
  getRandomValues: jest.fn((array: Uint8Array) => {
    for (let i = 0; i < array.length; i++) {
      array[i] = Math.floor(Math.random() * 256);
    }
    return array;
  }),
};

// 设置全局 crypto 对象
(global as any).crypto = cryptoMock;

// 模拟 CompressionStream 和 DecompressionStream
class MockCompressionStream {
  writable = {
    getWriter: () => ({
      write: jest.fn().mockResolvedValue(undefined),
      close: jest.fn().mockResolvedValue(undefined),
    }),
  };
  readable = {
    getReader: () => ({
      read: jest.fn()
        .mockResolvedValueOnce({ value: new Uint8Array([1, 2, 3]), done: false })
        .mockResolvedValueOnce({ value: undefined, done: true }),
    }),
  };
  constructor(_format: string) {}
}

class MockDecompressionStream {
  writable = {
    getWriter: () => ({
      write: jest.fn().mockResolvedValue(undefined),
      close: jest.fn().mockResolvedValue(undefined),
    }),
  };
  readable = {
    getReader: () => ({
      read: jest.fn()
        .mockResolvedValueOnce({ value: new Uint8Array([1, 2, 3]), done: false })
        .mockResolvedValueOnce({ value: undefined, done: true }),
    }),
  };
  constructor(_format: string) {}
}

(global as any).CompressionStream = MockCompressionStream;
(global as any).DecompressionStream = MockDecompressionStream;

// 模拟 atob 和 btoa
(global as any).atob = (str: string) => {
  return Buffer.from(str, 'base64').toString('binary');
};

(global as any).btoa = (str: string) => {
  return Buffer.from(str, 'binary').toString('base64');
};

// 清理每个测试后的模拟
afterEach(() => {
  jest.clearAllMocks();
});
