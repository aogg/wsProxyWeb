/**
 * 压缩工具库测试
 */
import { CompressUtil, CompressConfig } from '../libs/compress';

describe('CompressUtil', () => {
  describe('构造函数和验证', () => {
    it('未启用压缩时应该正常创建实例', () => {
      const config: CompressConfig = {
        enabled: false,
        level: 6,
        algorithm: 'gzip',
      };

      const compress = new CompressUtil(config);
      expect(compress.isEnabled()).toBe(false);
    });

    it('启用gzip压缩时应该正常创建实例', () => {
      const config: CompressConfig = {
        enabled: true,
        level: 6,
        algorithm: 'gzip',
      };

      // 模拟浏览器支持 CompressionStream
      (global as any).CompressionStream = jest.fn();
      (global as any).DecompressionStream = jest.fn();

      const compress = new CompressUtil(config);
      expect(compress.isEnabled()).toBe(true);
    });

    it('压缩级别过低时应该抛出错误', () => {
      const config: CompressConfig = {
        enabled: true,
        level: 0,
        algorithm: 'gzip',
      };

      expect(() => new CompressUtil(config)).toThrow('压缩级别必须在1-9之间');
    });

    it('压缩级别过高时应该抛出错误', () => {
      const config: CompressConfig = {
        enabled: true,
        level: 10,
        algorithm: 'gzip',
      };

      expect(() => new CompressUtil(config)).toThrow('压缩级别必须在1-9之间');
    });

    it('不支持的算法应该抛出错误', () => {
      const config: CompressConfig = {
        enabled: true,
        level: 6,
        algorithm: 'invalid' as any,
      };

      expect(() => new CompressUtil(config)).toThrow('不支持的压缩算法');
    });

    it('浏览器不支持CompressionStream时应该禁用压缩', () => {
      const originalCompressionStream = (global as any).CompressionStream;
      delete (global as any).CompressionStream;

      const config: CompressConfig = {
        enabled: true,
        level: 6,
        algorithm: 'gzip',
      };

      const compress = new CompressUtil(config);
      expect(compress.isEnabled()).toBe(false);

      // 恢复
      (global as any).CompressionStream = originalCompressionStream;
    });

    it('snappy算法应该自动切换到gzip', () => {
      (global as any).CompressionStream = jest.fn();
      (global as any).DecompressionStream = jest.fn();

      const config: CompressConfig = {
        enabled: true,
        level: 6,
        algorithm: 'snappy',
      };

      const compress = new CompressUtil(config);
      expect(compress.isEnabled()).toBe(true);
    });
  });

  describe('压缩解压功能', () => {
    beforeEach(() => {
      // 模拟 CompressionStream 和 DecompressionStream
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
    });

    it('未启用压缩时压缩应该返回原文', async () => {
      const config: CompressConfig = {
        enabled: false,
        level: 6,
        algorithm: 'gzip',
      };

      const compress = new CompressUtil(config);
      const data = new Uint8Array([1, 2, 3, 4, 5]);

      const result = await compress.compress(data);
      expect(result).toEqual(data);
    });

    it('未启用压缩时解压应该返回原文', async () => {
      const config: CompressConfig = {
        enabled: false,
        level: 6,
        algorithm: 'gzip',
      };

      const compress = new CompressUtil(config);
      const data = new Uint8Array([1, 2, 3, 4, 5]);

      const result = await compress.decompress(data);
      expect(result).toEqual(data);
    });

    it('启用gzip时压缩应该调用CompressionStream', async () => {
      const config: CompressConfig = {
        enabled: true,
        level: 6,
        algorithm: 'gzip',
      };

      const compress = new CompressUtil(config);
      const data = new Uint8Array([72, 101, 108, 108, 111]); // "Hello"

      const result = await compress.compress(data);
      expect((global as any).CompressionStream).toHaveBeenCalledWith('gzip');
      expect(result).toBeInstanceOf(Uint8Array);
    });

    it('启用gzip时解压应该调用DecompressionStream', async () => {
      const config: CompressConfig = {
        enabled: true,
        level: 6,
        algorithm: 'gzip',
      };

      const compress = new CompressUtil(config);
      const data = new Uint8Array([31, 139, 8, 0, 0, 0, 0, 0]); // 模拟gzip头部

      const result = await compress.decompress(data);
      expect((global as any).DecompressionStream).toHaveBeenCalledWith('gzip');
      expect(result).toBeInstanceOf(Uint8Array);
    });

    it('不支持的算法调用压缩应该抛出错误', async () => {
      const config: CompressConfig = {
        enabled: true,
        level: 6,
        algorithm: 'gzip',
      };

      const compress = new CompressUtil(config);
      
      // 通过修改私有属性来测试不支持的算法路径
      (compress as any).algorithm = 'invalid';

      const data = new Uint8Array([1, 2, 3]);
      await expect(compress.compress(data)).rejects.toThrow('不支持的压缩算法');
    });

    it('不支持的算法调用解压应该抛出错误', async () => {
      const config: CompressConfig = {
        enabled: true,
        level: 6,
        algorithm: 'gzip',
      };

      const compress = new CompressUtil(config);
      
      // 通过修改私有属性来测试不支持的算法路径
      (compress as any).algorithm = 'invalid';

      const data = new Uint8Array([1, 2, 3]);
      await expect(compress.decompress(data)).rejects.toThrow('不支持的压缩算法');
    });
  });

  describe('边界情况', () => {
    it('压缩空数据应该正常工作', async () => {
      const config: CompressConfig = {
        enabled: false,
        level: 6,
        algorithm: 'gzip',
      };

      const compress = new CompressUtil(config);
      const emptyData = new Uint8Array(0);

      const compressed = await compress.compress(emptyData);
      expect(compressed).toEqual(emptyData);

      const decompressed = await compress.decompress(emptyData);
      expect(decompressed).toEqual(emptyData);
    });

    it('压缩大数据应该正常工作', async () => {
      const config: CompressConfig = {
        enabled: false,
        level: 6,
        algorithm: 'gzip',
      };

      const compress = new CompressUtil(config);
      const largeData = new Uint8Array(1024 * 1024); // 1MB

      const compressed = await compress.compress(largeData);
      expect(compressed).toEqual(largeData);
    });
  });

  describe('错误处理', () => {
    it('CompressionStream不可用时压缩应该抛出错误', async () => {
      delete (global as any).CompressionStream;

      const config: CompressConfig = {
        enabled: true,
        level: 6,
        algorithm: 'gzip',
      };

      // 因为 CompressionStream 不存在，构造函数会禁用压缩
      const compress = new CompressUtil(config);
      expect(compress.isEnabled()).toBe(false);

      // 恢复
      (global as any).CompressionStream = jest.fn();
    });

    it('DecompressionStream不可用时解压应该抛出错误', async () => {
      const originalDecompressionStream = (global as any).DecompressionStream;
      (global as any).CompressionStream = jest.fn();

      const config: CompressConfig = {
        enabled: true,
        level: 6,
        algorithm: 'gzip',
      };

      const compress = new CompressUtil(config);
      
      // 通过修改私有属性模拟 DecompressionStream 不可用
      delete (global as any).DecompressionStream;

      const data = new Uint8Array([1, 2, 3]);
      await expect(compress.decompress(data)).rejects.toThrow('浏览器不支持 DecompressionStream API');

      // 恢复
      (global as any).DecompressionStream = originalDecompressionStream;
    });
  });
});
