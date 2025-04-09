package monitor

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// HealthStatus 健康状态
type HealthStatus struct {
	Component   string    `json:"component"`
	Status      string    `json:"status"`
	LastChecked time.Time `json:"last_checked"`
	Message     string    `json:"message,omitempty"`
}

// Monitor 监控系统
type Monitor struct {
	components map[string]*HealthStatus
	mutex      sync.RWMutex
	alertFunc  func(component, status, message string)
}

// NewMonitor 创建新的监控系统
func NewMonitor(alertFunc func(component, status, message string)) *Monitor {
	return &Monitor{
		components: make(map[string]*HealthStatus),
		alertFunc:  alertFunc,
	}
}

// RegisterComponent 注册组件
func (m *Monitor) RegisterComponent(component string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.components[component] = &HealthStatus{
		Component:   component,
		Status:      "unknown",
		LastChecked: time.Now(),
	}
}

// UpdateStatus 更新组件状态
func (m *Monitor) UpdateStatus(component, status, message string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if _, exists := m.components[component]; !exists {
		m.components[component] = &HealthStatus{
			Component: component,
		}
	}
	
	oldStatus := m.components[component].Status
	m.components[component].Status = status
	m.components[component].LastChecked = time.Now()
	m.components[component].Message = message
	
	// 如果状态变为不健康，触发告警
	if oldStatus != status && status != "healthy" && m.alertFunc != nil {
		m.alertFunc(component, status, message)
	}
}

// GetStatus 获取组件状态
func (m *Monitor) GetStatus(component string) *HealthStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if status, exists := m.components[component]; exists {
		return status
	}
	
	return nil
}

// GetAllStatus 获取所有组件状态
func (m *Monitor) GetAllStatus() []*HealthStatus {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	statuses := make([]*HealthStatus, 0, len(m.components))
	for _, status := range m.components {
		statuses = append(statuses, status)
	}
	
	return statuses
}

// CheckHTTPEndpoint 检查HTTP端点健康状态
func (m *Monitor) CheckHTTPEndpoint(component, url string) {
	resp, err := http.Get(url)
	
	if err != nil {
		m.UpdateStatus(component, "unhealthy", fmt.Sprintf("HTTP请求失败: %v", err))
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		m.UpdateStatus(component, "degraded", fmt.Sprintf("HTTP状态码非200: %d", resp.StatusCode))
		return
	}
	
	m.UpdateStatus(component, "healthy", "")
}

// StartChecking 开始定期检查
func (m *Monitor) StartChecking(component, url string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			m.CheckHTTPEndpoint(component, url)
		}
	}()
}