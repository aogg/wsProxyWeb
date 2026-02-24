// 性能优化工具库
package libs

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceConfig 性能配置
type PerformanceConfig struct {
	WorkerPoolSize     int `mapstructure:"workerPoolSize"`     // worker池大小（0表示不使用池，每个请求新开goroutine）
	RequestQueueSize   int `mapstructure:"requestQueueSize"`   // 请求队列大小（0表示不限制）
	MaxConcurrentConns int `mapstructure:"maxConcurrentConns"` // 最大并发连接处理数（0表示不限制）

	BufferPoolSize     int `mapstructure:"bufferPoolSize"`     // buffer池大小
	ChunkSize          int `mapstructure:"chunkSize"`          // 分块传输大小（字节），默认64KB
	EnableMetrics      bool `mapstructure:"enableMetrics"`      // 是否启用性能指标收集
	MetricsIntervalSec int `mapstructure:"metricsIntervalSec"` // 指标输出间隔（秒）
}

// PerformanceLib 性能优化库
type PerformanceLib struct {
	config    *PerformanceConfig
	workerPool *WorkerPool
	bufferPool *sync.Pool

	// 性能指标
	metrics *PerformanceMetrics
}

// PerformanceMetrics 性能指标
type PerformanceMetrics struct {
	TotalRequests       int64 // 总请求数
	ActiveRequests      int64 // 当前活跃请求数
	TotalConnections    int64 // 总连接数
	ActiveConnections   int64 // 当前活跃连接数
	RequestsPerSecond   int64 // 每秒请求数
	AvgResponseTimeMs   int64 // 平均响应时间（毫秒）
	BufferPoolHits      int64 // buffer池命中次数
	BufferPoolMisses    int64 // buffer池未命中次数
	PoolQueueLength     int64 // 池队列长度
}

// WorkerPool goroutine工作池
type WorkerPool struct {
	taskQueue chan TaskFunc
	workers   int
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// TaskFunc 任务函数类型
type TaskFunc func()

// BufferWrapper buffer包装器
type BufferWrapper struct {
	buf []byte
}

// NewPerformanceLib 创建性能优化库
func NewPerformanceLib(config *PerformanceConfig) *PerformanceLib {
	// 设置默认值
	if config.ChunkSize <= 0 {
		config.ChunkSize = 64 * 1024 // 64KB
	}
	if config.BufferPoolSize <= 0 {
		config.BufferPoolSize = 100
	}

	pl := &PerformanceLib{
		config:  config,
		metrics: &PerformanceMetrics{},
	}

	// 初始化buffer池
	pl.bufferPool = &sync.Pool{
		New: func() interface{} {
			atomic.AddInt64(&pl.metrics.BufferPoolMisses, 1)
			return &BufferWrapper{
				buf: make([]byte, 0, config.ChunkSize),
			}
		},
	}

	// 初始化worker池
	if config.WorkerPoolSize > 0 {
		pl.workerPool = NewWorkerPool(config.WorkerPoolSize, config.RequestQueueSize)
		Info("Worker池初始化完成: workers=%d, queueSize=%d", config.WorkerPoolSize, config.RequestQueueSize)
	}

	// 启动指标收集
	if config.EnableMetrics {
		go pl.startMetricsCollection()
	}

	Info("性能优化库初始化完成: chunkSize=%d, bufferPoolSize=%d, metrics=%v",
		config.ChunkSize, config.BufferPoolSize, config.EnableMetrics)

	return pl
}

// NewWorkerPool 创建worker池
func NewWorkerPool(workers int, queueSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	queueCap := queueSize
	if queueCap <= 0 {
		queueCap = 1000 // 默认队列大小
	}

	pool := &WorkerPool{
		taskQueue: make(chan TaskFunc, queueCap),
		workers:   workers,
		ctx:       ctx,
		cancel:    cancel,
	}

	// 启动workers
	for i := 0; i < workers; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

// worker 工作协程
func (p *WorkerPool) worker() {
	defer p.wg.Done()
	for {
		select {
		case task := <-p.taskQueue:
			if task != nil {
				task()
			}
		case <-p.ctx.Done():
			return
		}
	}
}

// Submit 提交任务
func (p *WorkerPool) Submit(task TaskFunc) bool {
	select {
	case p.taskQueue <- task:
		return true
	default:
		return false // 队列已满
	}
}

// SubmitWait 提交任务并等待完成
func (p *WorkerPool) SubmitWait(task TaskFunc) bool {
	select {
	case p.taskQueue <- task:
		return true
	case <-p.ctx.Done():
		return false
	}
}

// Stop 停止worker池
func (p *WorkerPool) Stop() {
	p.cancel()
	p.wg.Wait()
}

// GetQueueLength 获取队列长度
func (p *WorkerPool) GetQueueLength() int {
	return len(p.taskQueue)
}

// GetBuffer 从池中获取buffer
func (pl *PerformanceLib) GetBuffer() *BufferWrapper {
	atomic.AddInt64(&pl.metrics.BufferPoolHits, 1)
	return pl.bufferPool.Get().(*BufferWrapper)
}

// PutBuffer 将buffer放回池中
func (pl *PerformanceLib) PutBuffer(buf *BufferWrapper) {
	// 重置buffer
	buf.buf = buf.buf[:0]
	pl.bufferPool.Put(buf)
}

// SubmitTask 提交任务到worker池
func (pl *PerformanceLib) SubmitTask(task TaskFunc) bool {
	if pl.workerPool == nil {
		// 没有worker池，直接在新goroutine中执行
		go task()
		return true
	}

	atomic.AddInt64(&pl.metrics.PoolQueueLength, 1)
	success := pl.workerPool.Submit(func() {
		atomic.AddInt64(&pl.metrics.PoolQueueLength, -1)
		task()
	})

	if !success {
		atomic.AddInt64(&pl.metrics.PoolQueueLength, -1)
		// 队列满，在新goroutine中执行
		go task()
		return true
	}
	return success
}

// IncRequest 增加请求计数
func (pl *PerformanceLib) IncRequest() {
	atomic.AddInt64(&pl.metrics.TotalRequests, 1)
	atomic.AddInt64(&pl.metrics.ActiveRequests, 1)
}

// DecRequest 减少活跃请求计数
func (pl *PerformanceLib) DecRequest() {
	atomic.AddInt64(&pl.metrics.ActiveRequests, -1)
}

// IncConnection 增加连接计数
func (pl *PerformanceLib) IncConnection() {
	atomic.AddInt64(&pl.metrics.TotalConnections, 1)
	atomic.AddInt64(&pl.metrics.ActiveConnections, 1)
}

// DecConnection 减少活跃连接计数
func (pl *PerformanceLib) DecConnection() {
	atomic.AddInt64(&pl.metrics.ActiveConnections, -1)
}

// RecordResponseTime 记录响应时间
func (pl *PerformanceLib) RecordResponseTime(duration time.Duration) {
	// 使用简单的移动平均
	ms := duration.Milliseconds()
	for {
		old := atomic.LoadInt64(&pl.metrics.AvgResponseTimeMs)
		if old == 0 {
			atomic.StoreInt64(&pl.metrics.AvgResponseTimeMs, ms)
			break
		}
		// 移动平均: new = old * 0.9 + new * 0.1
		newAvg := old*9/10 + ms/10
		if atomic.CompareAndSwapInt64(&pl.metrics.AvgResponseTimeMs, old, newAvg) {
			break
		}
	}
}

// GetMetrics 获取性能指标
func (pl *PerformanceLib) GetMetrics() PerformanceMetrics {
	return PerformanceMetrics{
		TotalRequests:     atomic.LoadInt64(&pl.metrics.TotalRequests),
		ActiveRequests:    atomic.LoadInt64(&pl.metrics.ActiveRequests),
		TotalConnections:  atomic.LoadInt64(&pl.metrics.TotalConnections),
		ActiveConnections: atomic.LoadInt64(&pl.metrics.ActiveConnections),
		RequestsPerSecond: atomic.LoadInt64(&pl.metrics.RequestsPerSecond),
		AvgResponseTimeMs: atomic.LoadInt64(&pl.metrics.AvgResponseTimeMs),
		BufferPoolHits:    atomic.LoadInt64(&pl.metrics.BufferPoolHits),
		BufferPoolMisses:  atomic.LoadInt64(&pl.metrics.BufferPoolMisses),
		PoolQueueLength:   atomic.LoadInt64(&pl.metrics.PoolQueueLength),
	}
}

// startMetricsCollection 启动指标收集
func (pl *PerformanceLib) startMetricsCollection() {
	interval := time.Duration(pl.config.MetricsIntervalSec) * time.Second
	if interval <= 0 {
		interval = 30 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var lastRequests int64
	for range ticker.C {
		currentRequests := atomic.LoadInt64(&pl.metrics.TotalRequests)
		rps := currentRequests - lastRequests
		atomic.StoreInt64(&pl.metrics.RequestsPerSecond, rps)
		lastRequests = currentRequests

		m := pl.GetMetrics()
		Info("[性能指标] 总请求:%d, 活跃请求:%d, RPS:%d, 平均响应:%dms, Buffer池命中/未命中:%d/%d",
			m.TotalRequests, m.ActiveRequests, m.RequestsPerSecond, m.AvgResponseTimeMs,
			m.BufferPoolHits, m.BufferPoolMisses)
	}
}

// GetChunkSize 获取分块大小
func (pl *PerformanceLib) GetChunkSize() int {
	return pl.config.ChunkSize
}

// Stop 停止性能优化库
func (pl *PerformanceLib) Stop() {
	if pl.workerPool != nil {
		pl.workerPool.Stop()
	}
}

// DefaultPerformanceConfig 默认性能配置
func DefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		WorkerPoolSize:     0,   // 默认不使用worker池，每个请求独立goroutine
		RequestQueueSize:   1000,
		MaxConcurrentConns: 0,
		BufferPoolSize:     100,
		ChunkSize:          64 * 1024, // 64KB
		EnableMetrics:      true,
		MetricsIntervalSec: 30,
	}
}
