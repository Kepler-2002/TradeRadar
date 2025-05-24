package api

import (
    "TradeRadar/pkg/model"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
)

// NotificationService 通知服务
type NotificationService struct {
    webhookURL string
    client     *http.Client
}

// SendAlert 发送异动通知
func (ns *NotificationService) SendAlert(alert model.AlertEvent) error {
    message := ns.formatAlertMessage(alert)
    
    // 发送到用户的通知渠道（微信、邮件、APP推送等）
    return ns.sendToUser(alert.UserID, message)
}

// formatAlertMessage 格式化异动消息
func (ns *NotificationService) formatAlertMessage(alert model.AlertEvent) string {
    return fmt.Sprintf(`
🚨 股票异动提醒

📈 股票：%s (%s)
💰 当前价格：%.2f
📊 涨跌幅：%.2f%%
⚡ 异动类型：%s
🔥 严重程度：%s

🤖 AI分析建议：
%s

⏰ 时间：%s
`, alert.Data.Name, alert.Symbol, alert.Data.Price, alert.Data.ChangePercent, 
   alert.Type, alert.Severity, alert.AIAnalysis, alert.CreatedAt.Format("2006-01-02 15:04:05"))
}

// GenerateDailySummary 生成每日总结
func (ns *NotificationService) GenerateDailySummary(userID string, alerts []model.AlertEvent) string {
    if len(alerts) == 0 {
        return "今日您关注的股票表现平稳，无重大异动。"
    }
    
    summary := fmt.Sprintf("📊 今日股票异动总结 (%d条异动)\n\n", len(alerts))
    
    for _, alert := range alerts {
        summary += fmt.Sprintf("• %s: %s\n", alert.Symbol, alert.Message)
    }
    
    summary += "\n💡 建议：请结合市场整体情况和个人风险承受能力做出投资决策。"
    
    return summary
}