// 服务端安全控制逻辑
package logic

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"wsProxyWeb/server/src/types"
)

// SecurityLogicConfig 安全控制配置（注意：放在logic包内，避免与libs包产生循环依赖）
type SecurityLogicConfig struct {
	Enabled             bool
	MaxConnections      int
	MaxMessageBytes     int
	MaxRequestBodyBytes int

	RateLimitPerSecond float64
	RateBurst          int

	AllowIPs     []string
	DenyIPs      []string
	AllowDomains []string
	DenyDomains  []string
}

// SecurityLogic 安全控制逻辑类
type SecurityLogic struct {
	cfg SecurityLogicConfig

	mu          sync.Mutex
	tokenBucket map[string]*tokenBucketState // key: clientIP
}

type tokenBucketState struct {
	tokens float64
	last   time.Time
}

// NewSecurityLogic 创建安全控制逻辑实例
func NewSecurityLogic(cfg SecurityLogicConfig) *SecurityLogic {
	// 兜底默认值：避免配置缺失导致panic或“无限拒绝”
	if cfg.RateBurst <= 0 && cfg.RateLimitPerSecond > 0 {
		cfg.RateBurst = int(cfg.RateLimitPerSecond)
		if cfg.RateBurst < 1 {
			cfg.RateBurst = 1
		}
	}
	return &SecurityLogic{
		cfg:         cfg,
		tokenBucket: make(map[string]*tokenBucketState),
	}
}

// IsEnabled 是否启用安全控制
func (sl *SecurityLogic) IsEnabled() bool {
	return sl.cfg.Enabled
}

// CheckNewConnection 检查是否允许建立新连接
func (sl *SecurityLogic) CheckNewConnection(clientIP string, currentConnCount int) error {
	if !sl.cfg.Enabled {
		return nil
	}
	if sl.cfg.MaxConnections > 0 && currentConnCount >= sl.cfg.MaxConnections {
		return fmt.Errorf("连接数已达上限: %d", sl.cfg.MaxConnections)
	}
	if err := sl.checkIPAllowed(clientIP); err != nil {
		return err
	}
	return nil
}

// CheckRawMessageSize 检查原始消息大小（解密前）
func (sl *SecurityLogic) CheckRawMessageSize(rawBytes int) error {
	if !sl.cfg.Enabled {
		return nil
	}
	if sl.cfg.MaxMessageBytes > 0 && rawBytes > sl.cfg.MaxMessageBytes {
		return fmt.Errorf("消息过大: %d > %d", rawBytes, sl.cfg.MaxMessageBytes)
	}
	return nil
}

// CheckRequestMessage 检查request消息（URL/域名、请求体大小、频率限制等）
func (sl *SecurityLogic) CheckRequestMessage(clientIP string, msgData interface{}) error {
	if !sl.cfg.Enabled {
		return nil
	}

	if err := sl.checkIPAllowed(clientIP); err != nil {
		return err
	}

	reqData, err := sl.parseRequestData(msgData)
	if err != nil {
		return fmt.Errorf("解析请求数据失败: %v", err)
	}

	if err := sl.checkDomainAllowed(reqData.URL); err != nil {
		return err
	}

	if err := sl.checkRequestBodySize(reqData); err != nil {
		return err
	}

	if err := sl.checkRateLimit(clientIP); err != nil {
		return err
	}

	return nil
}

func (sl *SecurityLogic) parseRequestData(msgData interface{}) (*types.RequestData, error) {
	jsonData, err := json.Marshal(msgData)
	if err != nil {
		return nil, err
	}
	var reqData types.RequestData
	if err := json.Unmarshal(jsonData, &reqData); err != nil {
		return nil, err
	}
	return &reqData, nil
}

func (sl *SecurityLogic) checkRequestBodySize(reqData *types.RequestData) error {
	if sl.cfg.MaxRequestBodyBytes <= 0 {
		return nil
	}
	if reqData.Body == "" {
		return nil
	}

	// bodyEncoding默认按text处理
	encoding := strings.ToLower(strings.TrimSpace(reqData.BodyEncoding))
	if encoding == "" {
		encoding = "text"
	}

	var bodyBytes int
	if encoding == "base64" {
		// base64解码后的真实大小
		// 注意：这里仅做大小估算；如果base64内容非法，后续真正执行请求时会失败
		bodyBytes = base64.StdEncoding.DecodedLen(len(reqData.Body))
	} else {
		bodyBytes = len([]byte(reqData.Body))
	}

	if bodyBytes > sl.cfg.MaxRequestBodyBytes {
		return fmt.Errorf("请求体过大: %d > %d", bodyBytes, sl.cfg.MaxRequestBodyBytes)
	}
	return nil
}

func (sl *SecurityLogic) checkRateLimit(clientIP string) error {
	if sl.cfg.RateLimitPerSecond <= 0 {
		return nil
	}

	now := time.Now()

	sl.mu.Lock()
	defer sl.mu.Unlock()

	state, ok := sl.tokenBucket[clientIP]
	if !ok {
		// 初始给满桶，避免“刚连上就被限流”
		sl.tokenBucket[clientIP] = &tokenBucketState{
			tokens: float64(sl.cfg.RateBurst),
			last:   now,
		}
		state = sl.tokenBucket[clientIP]
	}

	elapsed := now.Sub(state.last).Seconds()
	if elapsed < 0 {
		elapsed = 0
	}

	// 补充token
	state.tokens += elapsed * sl.cfg.RateLimitPerSecond
	if state.tokens > float64(sl.cfg.RateBurst) {
		state.tokens = float64(sl.cfg.RateBurst)
	}
	state.last = now

	if state.tokens < 1 {
		return fmt.Errorf("请求过于频繁，请稍后再试")
	}
	state.tokens -= 1
	return nil
}

func (sl *SecurityLogic) checkIPAllowed(clientIP string) error {
	if clientIP == "" {
		// RemoteAddr解析失败时：宁可放行，避免误伤；可通过allowIPs做更严格控制
		return nil
	}

	// deny优先
	if ipInList(clientIP, sl.cfg.DenyIPs) {
		return fmt.Errorf("IP已被拒绝: %s", clientIP)
	}

	// allow非空时，必须命中allow
	if len(sl.cfg.AllowIPs) > 0 && !ipInList(clientIP, sl.cfg.AllowIPs) {
		return fmt.Errorf("IP不在允许列表: %s", clientIP)
	}

	return nil
}

func (sl *SecurityLogic) checkDomainAllowed(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL不能为空")
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL解析失败: %v", err)
	}
	host := strings.ToLower(strings.TrimSpace(u.Hostname()))
	if host == "" {
		return fmt.Errorf("URL缺少host")
	}

	// deny优先
	if domainInList(host, sl.cfg.DenyDomains) {
		return fmt.Errorf("域名已被拒绝: %s", host)
	}

	// allow非空时，必须命中allow
	if len(sl.cfg.AllowDomains) > 0 && !domainInList(host, sl.cfg.AllowDomains) {
		return fmt.Errorf("域名不在允许列表: %s", host)
	}

	return nil
}

func ipInList(clientIP string, patterns []string) bool {
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil {
		return false
	}

	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// CIDR
		if strings.Contains(p, "/") {
			_, cidr, err := net.ParseCIDR(p)
			if err != nil {
				continue
			}
			if cidr.Contains(ip) {
				return true
			}
			continue
		}
		// 单IP
		if ip.Equal(net.ParseIP(p)) {
			return true
		}
	}
	return false
}

func domainInList(host string, patterns []string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(host), "."))
	if host == "" {
		return false
	}
	for _, p := range patterns {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if p == "*" {
			return true
		}
		p = strings.TrimSuffix(p, ".")
		if host == p {
			return true
		}
		// 简单通配符：*.example.com
		if strings.HasPrefix(p, "*.") {
			suffix := strings.TrimPrefix(p, "*.")
			if suffix != "" && (host == suffix || strings.HasSuffix(host, "."+suffix)) {
				return true
			}
		}
	}
	return false
}

