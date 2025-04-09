package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	App struct {
		Name string `yaml:"name"`
		Env  string `yaml:"env"`
	} `yaml:"app"`
	
	DataSources struct {
		Tushare struct {
			APIKey   string        `yaml:"api_key"`
			BaseURL  string        `yaml:"base_url"`
			Timeout  time.Duration `yaml:"timeout"`
		} `yaml:"tushare"`
	} `yaml:"data_sources"`
	
	Database struct {
		TimescaleDB struct {
			Host     string `yaml:"host"`
			Port     int    `yaml:"port"`
			User     string `yaml:"user"`
			Password string `yaml:"password"`
			DBName   string `yaml:"dbname"`
			SSLMode  string `yaml:"sslmode"`
		} `yaml:"timescaledb"`
	} `yaml:"database"`
	
	NATS struct {
		URL       string `yaml:"url"`
		ClusterID string `yaml:"cluster_id"`
		ClientID  string `yaml:"client_id"`
	} `yaml:"nats"`
	
	API struct {
		Port         string        `yaml:"port"`
		ReadTimeout  time.Duration `yaml:"read_timeout"`
		WriteTimeout time.Duration `yaml:"write_timeout"`
	} `yaml:"api"`
}

// LoadConfig 从文件加载配置
func LoadConfig(path string) (*Config, error) {
	// 读取配置文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	
	// 解析YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	
	// 环境变量覆盖
	overrideFromEnv(&config)
	
	return &config, nil
}

// overrideFromEnv 使用环境变量覆盖配置
func overrideFromEnv(config *Config) {
	// 应用名称
	if env := os.Getenv("APP_NAME"); env != "" {
		config.App.Name = env
	}
	
	// 环境
	if env := os.Getenv("APP_ENV"); env != "" {
		config.App.Env = env
	}
	
	// Tushare配置
	if env := os.Getenv("TUSHARE_API_KEY"); env != "" {
		config.DataSources.Tushare.APIKey = env
	}
	if env := os.Getenv("TUSHARE_BASE_URL"); env != "" {
		config.DataSources.Tushare.BaseURL = env
	}
	
	// 数据库配置
	if env := os.Getenv("DB_HOST"); env != "" {
		config.Database.TimescaleDB.Host = env
	}
	if env := os.Getenv("DB_PORT"); env != "" {
		var port int
		fmt.Sscanf(env, "%d", &port)
		if port > 0 {
			config.Database.TimescaleDB.Port = port
		}
	}
	if env := os.Getenv("DB_USER"); env != "" {
		config.Database.TimescaleDB.User = env
	}
	if env := os.Getenv("DB_PASSWORD"); env != "" {
		config.Database.TimescaleDB.Password = env
	}
	if env := os.Getenv("DB_NAME"); env != "" {
		config.Database.TimescaleDB.DBName = env
	}
	
	// NATS配置
	if env := os.Getenv("NATS_URL"); env != "" {
		config.NATS.URL = env
	}
	if env := os.Getenv("NATS_CLUSTER_ID"); env != "" {
		config.NATS.ClusterID = env
	}
	if env := os.Getenv("NATS_CLIENT_ID"); env != "" {
		config.NATS.ClientID = env
	}
	
	// API配置
	if env := os.Getenv("API_PORT"); env != "" {
		config.API.Port = env
	}
}

// GetDefaultConfigPath 获取默认配置文件路径
func GetDefaultConfigPath() string {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev" // 默认开发环境
	}
	
	return fmt.Sprintf("configs/%s/app.yaml", env)
}