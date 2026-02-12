package tests

import (
	"testing"

	"wsProxyWeb/server/src/logic"
)

func TestSecurityLogic_域名通配符允许(t *testing.T) {
	sl := prepareSecurityLogic(logic.SecurityLogicConfig{AllowDomains: []string{"*.example.com"}})
	if err := sl.CheckRequestMessage("127.0.0.1", prepareRequestMsgData("https://api.example.com/a")); err != nil {
		t.Fatalf("期望允许，但被拒绝: %v", err)
	}
}

func TestSecurityLogic_IP_CIDR_拒绝优先(t *testing.T) {
	sl := prepareSecurityLogic(logic.SecurityLogicConfig{AllowIPs: []string{"10.0.0.0/8"}, DenyIPs: []string{"10.1.2.3"}})
	if err := sl.CheckNewConnection("10.1.2.3", 0); err == nil {
		t.Fatalf("期望拒绝，但被允许")
	}
}

