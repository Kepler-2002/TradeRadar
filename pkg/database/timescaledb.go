package database

import (
	"TradeRadar/pkg/config"
	"TradeRadar/pkg/model"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"time"
)

type TimescaleDB struct {
	db *gorm.DB
}

func NewTimescaleDB(cfg *config.Config) (*TimescaleDB, error) {
	dbCfg := cfg.Database.TimescaleDB

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Password, dbCfg.DBName, dbCfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层 sql.DB 对象设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// 自动迁移
	err = db.AutoMigrate(
		&model.User{},
		&model.Stock{},
		&model.StockQuote{},
		&model.Subscription{},
		&model.AlertEvent{},
		&model.NewsEvent{},
	)
	if err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return &TimescaleDB{db: db}, nil
}

func (t *TimescaleDB) Close() error {
	sqlDB, err := t.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
