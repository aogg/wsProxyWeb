package tests

import "wsProxyWeb/server/src/logic"

// 准备：构造一个默认的SecurityLogic，测试用例按需覆盖配置项
func prepareSecurityLogic(cfg logic.SecurityLogicConfig) *logic.SecurityLogic {
	// 默认启用，避免每个用例都写Enabled=true
	if !cfg.Enabled {
		cfg.Enabled = true
	}
	if cfg.RateLimitPerSecond == 0 {
		cfg.RateLimitPerSecond = 1000
	}
	if cfg.RateBurst == 0 {
		cfg.RateBurst = 1000
	}
	return logic.NewSecurityLogic(cfg)
}

func prepareRequestMsgData(rawURL string) map[string]interface{} {
	return map[string]interface{}{
		"url":          rawURL,
		"method":       "GET",
		"headers":      map[string]string{},
		"body":         "",
		"bodyEncoding": "text",
	}
}

