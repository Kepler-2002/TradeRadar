package scheduler

import (
	"log"

	"github.com/dewei/TradeRadar/pkg/engine"
	"github.com/dewei/TradeRadar/pkg/repository"
	"github.com/robfig/cron/v3"
)

// Scheduler 任务调度器
type Scheduler struct {
	cron   *cron.Cron
	engine *engine.RuleEngine
}

// NewScheduler 创建任务调度器
func NewScheduler(engine *engine.RuleEngine) *Scheduler {
	return &Scheduler{
		cron:   cron.New(),
		engine: engine,
	}
}

// Start 启动调度器
func (s *Scheduler) Start() {
	// 每日开盘前加载规则
	s.cron.AddFunc("0 30 9 * * 1-5", func() {
		log.Println("加载交易规则...")
		rules := repository.LoadActiveRules()
		s.engine.ReloadRules(rules)
	})

	// 每5分钟检查数据源健康状态
	s.cron.AddFunc("@every 5m", s.monitorDataHealth)

	s.cron.Start()
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.cron.Stop()
}

// monitorDataHealth 监控数据源健康状态
func (s *Scheduler) monitorDataHealth() {
	log.Println("检查数据源健康状态...")
	// 实际实现中需要检查各数据源连接状态
}