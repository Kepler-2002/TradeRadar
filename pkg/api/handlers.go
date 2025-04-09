package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/dewei/TradeRadar/pkg/collector"
	"github.com/dewei/TradeRadar/pkg/engine"
	"github.com/dewei/TradeRadar/pkg/model"
	"github.com/dewei/TradeRadar/pkg/repository"
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
	UserID  string              `json:"user_id" binding:"required"`
	Symbols []string            `json:"symbols" binding:"required"`
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
		"status": "success",
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