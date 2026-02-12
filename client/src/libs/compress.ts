// 压缩工具库

// 压缩配置接口
export interface CompressConfig {
  enabled: boolean;
  level: number; // 压缩级别：1-9
  algorithm: 'gzip' | 'snappy';
}

// 压缩工具类
export class CompressUtil {
  private enabled: boolean;
  private level: number;
  private algorithm: string;

  constructor(config: CompressConfig) {
    this.enabled = config.enabled;
    this.level = config.level;
    this.algorithm = config.algorithm;

    // 验证压缩级别
    if (config.level < 1 || config.level > 9) {
      throw new Error(`压缩级别必须在1-9之间，当前值: ${config.level}`);
    }

    // 验证算法
    if (config.algorithm !== 'gzip' && config.algorithm !== 'snappy') {
      throw new Error(`不支持的压缩算法: ${config.algorithm}，支持: gzip, snappy`);
    }

    // 检查浏览器支持
    if (config.enabled && config.algorithm === 'gzip') {
      if (typeof CompressionStream === 'undefined') {
        console.warn('浏览器不支持 CompressionStream API，压缩功能将被禁用');
        this.enabled = false;
      }
    } else if (config.enabled && config.algorithm === 'snappy') {
      console.warn('浏览器不支持 snappy 压缩，将使用 gzip 替代');
      this.algorithm = 'gzip';
    }
  }

  /**
   * 压缩数据
   */
  async compress(data: Uint8Array): Promise<Uint8Array> {
    if (!this.enabled) {
      return data;
    }

    if (this.algorithm === 'gzip') {
      return this.compressGzip(data);
    } else {
      throw new Error(`不支持的压缩算法: ${this.algorithm}`);
    }
  }

  /**
   * 解压数据
   */
  async decompress(data: Uint8Array): Promise<Uint8Array> {
    if (!this.enabled) {
      return data;
    }

    if (this.algorithm === 'gzip') {
      return this.decompressGzip(data);
    } else {
      throw new Error(`不支持的压缩算法: ${this.algorithm}`);
    }
  }

  /**
   * 使用gzip压缩
   * 注意：CompressionStream API 不支持自定义压缩级别，使用默认级别
   */
  private async compressGzip(data: Uint8Array): Promise<Uint8Array> {
    if (typeof CompressionStream === 'undefined') {
      throw new Error('浏览器不支持 CompressionStream API');
    }

    try {
      // 创建压缩流
      const stream = new CompressionStream('gzip');
      const writer = stream.writable.getWriter();
      const reader = stream.readable.getReader();

      // 写入数据
      writer.write(data as unknown as BufferSource);
      writer.close();

      // 读取压缩后的数据
      const chunks: Uint8Array[] = [];
      let done = false;

      while (!done) {
        const { value, done: readerDone } = await reader.read();
        done = readerDone;
        if (value) {
          chunks.push(value);
        }
      }

      // 合并所有块
      const totalLength = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
      const result = new Uint8Array(totalLength);
      let offset = 0;
      for (const chunk of chunks) {
        result.set(chunk, offset);
        offset += chunk.length;
      }

      return result;
    } catch (error) {
      throw new Error(`gzip压缩失败: ${error}`);
    }
  }

  /**
   * 使用gzip解压
   */
  private async decompressGzip(data: Uint8Array): Promise<Uint8Array> {
    if (typeof DecompressionStream === 'undefined') {
      throw new Error('浏览器不支持 DecompressionStream API');
    }

    try {
      // 创建解压流
      const stream = new DecompressionStream('gzip');
      const writer = stream.writable.getWriter();
      const reader = stream.readable.getReader();

      // 写入数据
      writer.write(data as unknown as BufferSource);
      writer.close();

      // 读取解压后的数据
      const chunks: Uint8Array[] = [];
      let done = false;

      while (!done) {
        const { value, done: readerDone } = await reader.read();
        done = readerDone;
        if (value) {
          chunks.push(value);
        }
      }

      // 合并所有块
      const totalLength = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
      const result = new Uint8Array(totalLength);
      let offset = 0;
      for (const chunk of chunks) {
        result.set(chunk, offset);
        offset += chunk.length;
      }

      return result;
    } catch (error) {
      throw new Error(`gzip解压失败: ${error}`);
    }
  }

  /**
   * 检查压缩是否启用
   */
  isEnabled(): boolean {
    return this.enabled;
  }
}
