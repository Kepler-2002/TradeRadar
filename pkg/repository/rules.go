package repository

import (
	"TradeRadar/pkg/model"
)

// LoadActiveRules 加载活跃规则
// 实际项目中应该从数据库加载
func LoadActiveRules() map[string][]model.DetectionRule {
	// 模拟从数据库加载规则
	rules := make(map[string][]model.DetectionRule)
	
	// 全局规则
	rules["*"] = []model.DetectionRule{
		{
			Type:      model.AlertPriceVolatility,
			Threshold: 5.0,
		},
		{
			Type:      model.AlertVolumeSpike,
			Threshold: 1000000,
		},
	}
	
	// 特定股票规则
	rules["000001.SZ"] = []model.DetectionRule{
		{
			Type:      model.AlertPriceVolatility,
			Threshold: 3.0, // 更敏感的阈值
		},
	}
	
	return rules
}