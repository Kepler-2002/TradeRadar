package api

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// Server API服务器
type Server struct {
	router *gin.Engine
	srv    *http.Server
}

// NewServer 创建新的API服务器
func NewServer(port string) *Server {
	router := gin.Default()
	
	// 设置中间件
	router.Use(gin.Recovery())
	router.Use(gin.Logger())
	
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}
	
	return &Server{
		router: router,
		srv:    srv,
	}
}

// SetupRoutes 设置路由
func (s *Server) SetupRoutes(handlers *Handlers) {
	// 健康检查
	s.router.GET("/health", handlers.HealthCheck)
	s.router.GET("/ready", handlers.ReadinessCheck)
	
	// API v1 路由组
	v1 := s.router.Group("/api/v1")
	{
		// 行情接口
		v1.GET("/quotes", handlers.GetQuotes)
		
		// 异动订阅接口
		v1.POST("/alerts/subscribe", handlers.SubscribeAlerts)
		
		// 异动历史接口
		v1.GET("/alerts/history", handlers.GetAlertHistory)
	}
}

// Start 启动服务器
func (s *Server) Start() {
	// 在goroutine中启动服务器
	go func() {
		log.Printf("API服务器启动在 %s\n", s.srv.Addr)
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v\n", err)
		}
	}()
	
	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务器...")
	
	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	// 优雅关闭
	if err := s.srv.Shutdown(ctx); err != nil {
		log.Fatalf("服务器关闭失败: %v\n", err)
	}
	
	log.Println("服务器已关闭")
}