// pkg/database/stock.go
package database

import (
	"TradeRadar/pkg/model"
	"fmt"
	"gorm.io/gorm"
)

type StockDB struct {
	db *gorm.DB
}

func (t *TimescaleDB) Stock() *StockDB {
	return &StockDB{db: t.db}
}

func (s *StockDB) Save(stock *model.Stock) error {
	return s.db.Save(stock).Error
}

func (s *StockDB) SaveBatch(stocks []*model.Stock) error {
	return s.db.CreateInBatches(stocks, 500).Error
}

func (s *StockDB) GetBySymbol(symbol string) (*model.Stock, error) {
	var stock model.Stock
	err := s.db.First(&stock, "symbol = ?", symbol).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("股票不存在")
		}
		return nil, fmt.Errorf("获取股票信息失败: %w", err)
	}
	return &stock, nil
}

func (s *StockDB) GetByMarket(market string, limit int) ([]*model.Stock, error) {
	var stocks []*model.Stock
	err := s.db.Where("market = ? AND is_active = ?", market, true).
		Limit(limit).
		Find(&stocks).Error

	if err != nil {
		return nil, fmt.Errorf("查询市场股票失败: %w", err)
	}
	return stocks, nil
}

func (s *StockDB) GetByIndustry(industry string, limit int) ([]*model.Stock, error) {
	var stocks []*model.Stock
	err := s.db.Where("industry = ? AND is_active = ?", industry, true).
		Limit(limit).
		Find(&stocks).Error

	if err != nil {
		return nil, fmt.Errorf("查询行业股票失败: %w", err)
	}
	return stocks, nil
}

func (s *StockDB) GetBySector(sector string, limit int) ([]*model.Stock, error) {
	var stocks []*model.Stock
	err := s.db.Where("sector = ? AND is_active = ?", sector, true).
		Limit(limit).
		Find(&stocks).Error

	if err != nil {
		return nil, fmt.Errorf("查询板块股票失败: %w", err)
	}
	return stocks, nil
}

func (s *StockDB) Search(keyword string, limit int) ([]*model.Stock, error) {
	var stocks []*model.Stock
	searchPattern := "%" + keyword + "%"

	err := s.db.Where("(symbol ILIKE ? OR name ILIKE ?) AND is_active = ?",
		searchPattern, searchPattern, true).
		Limit(limit).
		Find(&stocks).Error

	if err != nil {
		return nil, fmt.Errorf("搜索股票失败: %w", err)
	}
	return stocks, nil
}

func (s *StockDB) GetActiveStocks() ([]*model.Stock, error) {
	var stocks []*model.Stock
	err := s.db.Where("is_active = ?", true).Find(&stocks).Error
	if err != nil {
		return nil, fmt.Errorf("查询活跃股票失败: %w", err)
	}
	return stocks, nil
}

func (s *StockDB) UpdateStatus(symbol string, isActive bool) error {
	return s.db.Model(&model.Stock{}).
		Where("symbol = ?", symbol).
		Update("is_active", isActive).Error
}

func (s *StockDB) ExistsBySymbol(symbol string) (bool, error) {
	var count int64
	err := s.db.Model(&model.Stock{}).Where("symbol = ?", symbol).Count(&count).Error
	return count > 0, err
}
