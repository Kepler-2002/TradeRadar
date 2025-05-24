package model

import "time"

// DetectionRule 异动检测规则（引擎内部使用）
type DetectionRule struct {
	ID          string    `json:"id"`
	Type        AlertType `json:"type"`        // 检测类型：价格异动、成交量异动、新闻异动
	Threshold   float64   `json:"threshold"`   // 触发阈值
	Enabled     bool      `json:"enabled"`     // 是否启用
	Description string    `json:"description"` // 规则描述
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DetectionCondition 检测条件（可扩展）
type DetectionCondition struct {
	Field    string      `json:"field"`    // 检测字段：price, volume, change_percent 等
	Operator string      `json:"operator"` // 操作符：>, <, >=, <=, =, !=
	Value    interface{} `json:"value"`    // 比较值
}

// RuleTemplate 规则模板
type RuleTemplate struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Type        AlertType            `json:"type"`
	Conditions  []DetectionCondition `json:"conditions"`
	Description string               `json:"description"`
	IsBuiltIn   bool                 `json:"is_built_in"` // 是否为内置模板
	CreatedAt   time.Time            `json:"created_at"`
}
