package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"TradeRadar/pkg/config"
	"TradeRadar/pkg/monitor"
)

func main() {
	log.Println("启动监控服务...")

	// 加载配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = config.GetDefaultConfigPath()
	}
	
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v\n", err)
	}
	
	// 创建监控系统
	mon := monitor.NewMonitor(func(component, status, message string) {
		log.Printf("告警: 组件[%s]状态变为[%s], 消息: %s\n", component, status, message)
		// 实际项目中可以发送邮件、短信等
	})
	
	// 注册组件
	mon.RegisterComponent("api-service")
	mon.RegisterComponent("collector-service")
	mon.RegisterComponent("engine-service")
	mon.RegisterComponent("database")
	mon.RegisterComponent("nats")
	
	// 开始定期检查
	apiPort := cfg.API.Port
	if apiPort == "" {
		apiPort = "8080"
	}
	mon.StartChecking("api-service", fmt.Sprintf("http://localhost:%s/health", apiPort), 30*time.Second)
	
	// 设置HTTP处理程序
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	
	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		statuses := mon.GetAllStatus()
		json.NewEncoder(w).Encode(statuses)
	})
	
	// 启动HTTP服务器
	port := "8081" // 监控服务端口
	log.Printf("监控服务启动在 :%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("启动HTTP服务器失败: %v\n", err)
	}
}