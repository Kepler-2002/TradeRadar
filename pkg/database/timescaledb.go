
package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v4/stdlib"
	"TradeRadar/pkg/config"
)

// TimescaleDB TimescaleDB数据库连接
type TimescaleDB struct {
	db *sql.DB
}

// NewTimescaleDB 创建新的TimescaleDB连接
func NewTimescaleDB(cfg *config.Config) (*TimescaleDB, error) {
	dbCfg := cfg.Database.TimescaleDB
	
	// 构建连接字符串
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbCfg.Host, dbCfg.Port, dbCfg.User, dbCfg.Password, dbCfg.DBName, dbCfg.SSLMode,
	)
	
	// 连接数据库
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}
	
	// 设置连接池参数
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	
	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("测试数据库连接失败: %w", err)
	}
	
	return &TimescaleDB{db: db}, nil
}

// Close 关闭数据库连接
func (t *TimescaleDB) Close() error {
	return t.db.Close()
}

// SaveQuote 保存行情数据
func (t *TimescaleDB) SaveQuote(symbol string, price, volume float64, timestamp time.Time) error {
	query := `
		INSERT INTO stock_quotes (symbol, price, volume, timestamp)
		VALUES ($1, $2, $3, $4)
	`
	
	_, err := t.db.Exec(query, symbol, price, volume, timestamp)
	if err != nil {
		return fmt.Errorf("保存行情数据失败: %w", err)
	}
	
	return nil
}

// SaveAlert 保存异动事件
func (t *TimescaleDB) SaveAlert(symbol string, alertType int, intensity float64, timestamp time.Time) error {
	query := `
		INSERT INTO stock_alerts (symbol, alert_type, intensity, timestamp)
		VALUES ($1, $2, $3, $4)
	`
	
	_, err := t.db.Exec(query, symbol, alertType, intensity, timestamp)
	if err != nil {
		return fmt.Errorf("保存异动事件失败: %w", err)
	}
	
	return nil
}

// GetRecentAlerts 获取最近的异动事件
func (t *TimescaleDB) GetRecentAlerts(symbol string, limit int) ([]map[string]interface{}, error) {
	var query string
	var args []interface{}
	
	if symbol == "" {
		query = `
			SELECT symbol, alert_type, intensity, timestamp
			FROM stock_alerts
			ORDER BY timestamp DESC
			LIMIT $1
		`
		args = []interface{}{limit}
	} else {
		query = `
			SELECT symbol, alert_type, intensity, timestamp
			FROM stock_alerts
			WHERE symbol = $1
			ORDER BY timestamp DESC
			LIMIT $2
		`
		args = []interface{}{symbol, limit}
	}
	
	rows, err := t.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询异动事件失败: %w", err)
	}
	defer rows.Close()
	
	var results []map[string]interface{}
	
	for rows.Next() {
		var symbol string
		var alertType int
		var intensity float64
		var timestamp time.Time
		
		if err := rows.Scan(&symbol, &alertType, &intensity, &timestamp); err != nil {
			return nil, fmt.Errorf("扫描行数据失败: %w", err)
		}
		
		result := map[string]interface{}{
			"symbol":    symbol,
			"type":      alertType,
			"intensity": intensity,
			"timestamp": timestamp,
		}
		
		results = append(results, result)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("迭代行数据失败: %w", err)
	}
	
	return results, nil
}