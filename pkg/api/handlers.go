package api

import (
	"fmt"
	_ "math"
	"net/http"
	"strings"
	"time"

	"TradeRadar/pkg/collector"
	"TradeRadar/pkg/engine"
	"TradeRadar/pkg/model"
	"TradeRadar/pkg/repository"
	"github.com/gin-gonic/gin"
)

// Handlers API处理程序
type Handlers struct {
	quoteFetcher collector.QuoteFetcher
	ruleEngine   *engine.RuleEngine
	repository   *repository.Repository
}

// NewHandlers 创建新的API处理程序
func NewHandlers(
	quoteFetcher collector.QuoteFetcher,
	ruleEngine *engine.RuleEngine,
	repository *repository.Repository,
) *Handlers {
	return &Handlers{
		quoteFetcher: quoteFetcher,
		ruleEngine:   ruleEngine,
		repository:   repository,
	}
}

// HealthCheck 健康检查处理程序
func (h *Handlers) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// ReadinessCheck 就绪检查处理程序
func (h *Handlers) ReadinessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// GetQuotes 获取行情处理程序
func (h *Handlers) GetQuotes(c *gin.Context) {
	// 获取请求参数
	symbolsParam := c.Query("symbols")
	if symbolsParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbols参数不能为空",
		})
		return
	}

	// 分割股票代码
	symbols := strings.Split(symbolsParam, ",")

	// 获取行情数据
	quotes, err := h.quoteFetcher.FetchRealtime(symbols)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取行情数据失败: " + err.Error(),
		})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"data": quotes,
	})
}

// SubscribeRequest 订阅请求
type SubscribeRequest struct {
	UserID  string                `json:"user_id" binding:"required"`
	Symbols []string              `json:"symbols" binding:"required"`
	Rules   []model.DetectionRule `json:"rules" binding:"required"`
}

// SubscribeAlerts 订阅异动处理程序
func (h *Handlers) SubscribeAlerts(c *gin.Context) {
	var req SubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的请求参数: " + err.Error(),
		})
		return
	}

	// 保存订阅
	err := h.repository.SaveSubscription(req.UserID, req.Symbols, req.Rules)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存订阅失败: " + err.Error(),
		})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "订阅成功",
	})
}

// GetAlertHistory 获取异动历史处理程序
func (h *Handlers) GetAlertHistory(c *gin.Context) {
	// 获取请求参数
	symbol := c.Query("symbol")
	limit := 10 // 默认限制

	// 获取异动历史
	alerts, err := h.repository.GetAlertHistory(symbol, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取异动历史失败: " + err.Error(),
		})
		return
	}

	// 返回结果
	c.JSON(http.StatusOK, gin.H{
		"data": alerts,
	})
}

// SubscribeStockRequest 订阅股票请求
type SubscribeStockRequest struct {
	UserID     string            `json:"user_id" binding:"required"`
	Symbols    []string          `json:"symbols" binding:"required"`
	AlertRules []model.AlertRule `json:"alert_rules,omitempty"`
}

// SubscribeStock 订阅股票接口
func (h *Handlers) SubscribeStock(c *gin.Context) {
	var req SubscribeStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证股票代码有效性
	validSymbols, err := h.validateSymbols(req.Symbols)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的股票代码: " + err.Error()})
		return
	}

	// 保存订阅信息到数据库
	subscription := &model.Subscription{
		UserID:     req.UserID,
		Symbols:    validSymbols,
		AlertRules: req.AlertRules,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Status:     "active",
	}

	if err := h.repository.SaveSubscriptionModel(subscription); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存订阅失败: " + err.Error()})
		return
	}

	// 通知监控引擎开始监控这些股票
	h.notifyEngine(subscription)

	c.JSON(http.StatusOK, gin.H{
		"message":         "订阅成功",
		"subscription_id": subscription.ID,
		"symbols":         validSymbols,
	})
}

// validateSymbols 验证股票代码
func (h *Handlers) validateSymbols(symbols []string) ([]string, error) {
	// 调用数据源验证股票代码是否存在
	var validSymbols []string
	for _, symbol := range symbols {
		// 简单的股票代码格式验证
		if len(symbol) >= 6 && (strings.HasSuffix(symbol, ".SZ") || strings.HasSuffix(symbol, ".SH")) {
			validSymbols = append(validSymbols, symbol)
		}
	}
	if len(validSymbols) == 0 {
		return nil, fmt.Errorf("没有有效的股票代码")
	}
	return validSymbols, nil
}

// notifyEngine 通知监控引擎
func (h *Handlers) notifyEngine(subscription *model.Subscription) {
	// 这里应该通知监控引擎开始监控新的订阅
	// 可以通过消息队列或直接调用引擎方法
	if h.ruleEngine != nil {
		h.ruleEngine.AddSubscription(subscription)
	}
}

// GetUserSubscriptions 获取用户订阅列表
func (h *Handlers) GetUserSubscriptions(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id参数不能为空"})
		return
	}

	subscriptions, err := h.repository.GetUserSubscriptions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取订阅列表失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": subscriptions,
	})
}

// UpdateSubscription 更新订阅
func (h *Handlers) UpdateSubscription(c *gin.Context) {
	subscriptionID := c.Param("id")
	if subscriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订阅ID不能为空"})
		return
	}

	var req SubscribeStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证股票代码有效性
	validSymbols, err := h.validateSymbols(req.Symbols)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的股票代码: " + err.Error()})
		return
	}

	subscription := &model.Subscription{
		ID:         subscriptionID,
		UserID:     req.UserID,
		Symbols:    validSymbols,
		AlertRules: req.AlertRules,
		UpdatedAt:  time.Now(),
		Status:     "active",
	}

	if err := h.repository.UpdateSubscription(subscription); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新订阅失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "订阅更新成功",
	})
}

// DeleteSubscription 删除订阅
func (h *Handlers) DeleteSubscription(c *gin.Context) {
	subscriptionID := c.Param("id")
	if subscriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "订阅ID不能为空"})
		return
	}

	if err := h.repository.DeleteSubscription(subscriptionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除订阅失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "订阅删除成功",
	})
}
