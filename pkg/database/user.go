// pkg/database/user.go
package database

import (
	"TradeRadar/pkg/model"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type UserDB struct {
	db *gorm.DB
}

func (t *TimescaleDB) User() *UserDB {
	return &UserDB{db: t.db}
}

func (u *UserDB) Save(user *model.User) error {
	return u.db.Save(user).Error
}

func (u *UserDB) Create(user *model.User) error {
	return u.db.Create(user).Error
}

func (u *UserDB) GetByID(userID string) (*model.User, error) {
	var user model.User
	err := u.db.First(&user, "id = ?", userID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}
	return &user, nil
}

func (u *UserDB) GetByUsername(username string) (*model.User, error) {
	var user model.User
	err := u.db.First(&user, "username = ?", username).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("根据用户名获取用户信息失败: %w", err)
	}
	return &user, nil
}

func (u *UserDB) GetByEmail(email string) (*model.User, error) {
	var user model.User
	err := u.db.First(&user, "email = ?", email).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("根据邮箱获取用户信息失败: %w", err)
	}
	return &user, nil
}

func (u *UserDB) GetByPhone(phone string) (*model.User, error) {
	var user model.User
	err := u.db.First(&user, "phone = ?", phone).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("根据手机号获取用户信息失败: %w", err)
	}
	return &user, nil
}

func (u *UserDB) UpdateLastLogin(userID string) error {
	return u.db.Model(&model.User{}).
		Where("id = ?", userID).
		Update("last_login_at", time.Now()).Error
}

func (u *UserDB) UpdateProfile(userID string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()
	result := u.db.Model(&model.User{}).
		Where("id = ?", userID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("更新用户信息失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("用户不存在或无更新")
	}

	return nil
}

func (u *UserDB) UpdateAvatar(userID string, avatarURL string) error {
	return u.db.Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"avatar":     avatarURL,
			"updated_at": time.Now(),
		}).Error
}

func (u *UserDB) UpdateStatus(userID string, status int) error {
	return u.db.Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

func (u *UserDB) UpdatePassword(userID string, passwordHash string) error {
	// 假设 User model 中有 PasswordHash 字段
	return u.db.Model(&model.User{}).
		Where("id = ?", userID).
		Updates(map[string]interface{}{
			"password_hash": passwordHash,
			"updated_at":    time.Now(),
		}).Error
}

func (u *UserDB) Delete(userID string) error {
	// 软删除
	return u.db.Delete(&model.User{}, "id = ?", userID).Error
}

func (u *UserDB) GetActiveUsers(limit, offset int) ([]*model.User, error) {
	var users []*model.User
	err := u.db.Where("status = ?", 1). // 假设 1 为活跃状态
						Limit(limit).
						Offset(offset).
						Find(&users).Error

	if err != nil {
		return nil, fmt.Errorf("查询活跃用户失败: %w", err)
	}
	return users, nil
}

func (u *UserDB) GetRecentUsers(days int, limit int) ([]*model.User, error) {
	var users []*model.User
	startTime := time.Now().AddDate(0, 0, -days)

	err := u.db.Where("created_at >= ?", startTime).
		Order("created_at DESC").
		Limit(limit).
		Find(&users).Error

	if err != nil {
		return nil, fmt.Errorf("查询最近注册用户失败: %w", err)
	}
	return users, nil
}

func (u *UserDB) GetUserWithSubscriptions(userID string) (*model.User, error) {
	var user model.User
	err := u.db.Preload("Subscriptions", "status != ?", model.SubscriptionStatusCancelled).
		First(&user, "id = ?", userID).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("获取用户及订阅信息失败: %w", err)
	}
	return &user, nil
}

func (u *UserDB) GetUserWithAlerts(userID string, limit int) (*model.User, error) {
	var user model.User
	err := u.db.Preload("Alerts", func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at DESC").Limit(limit)
	}).First(&user, "id = ?", userID).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("获取用户及异动信息失败: %w", err)
	}
	return &user, nil
}

func (u *UserDB) Search(keyword string, limit int) ([]*model.User, error) {
	var users []*model.User
	searchPattern := "%" + keyword + "%"

	err := u.db.Where("username ILIKE ? OR nickname ILIKE ? OR email ILIKE ?",
		searchPattern, searchPattern, searchPattern).
		Limit(limit).
		Find(&users).Error

	if err != nil {
		return nil, fmt.Errorf("搜索用户失败: %w", err)
	}
	return users, nil
}

func (u *UserDB) ExistsByUsername(username string) (bool, error) {
	var count int64
	err := u.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func (u *UserDB) ExistsByEmail(email string) (bool, error) {
	var count int64
	err := u.db.Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (u *UserDB) ExistsByPhone(phone string) (bool, error) {
	var count int64
	err := u.db.Model(&model.User{}).Where("phone = ?", phone).Count(&count).Error
	return count > 0, err
}

func (u *UserDB) GetUserStats(userID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 获取订阅数量
	var subscriptionCount int64
	err := u.db.Model(&model.Subscription{}).
		Where("user_id = ? AND status != ?", userID, model.SubscriptionStatusCancelled).
		Count(&subscriptionCount).Error
	if err != nil {
		return nil, fmt.Errorf("统计订阅数量失败: %w", err)
	}

	// 获取异动事件数量
	var alertCount int64
	err = u.db.Model(&model.AlertEvent{}).
		Where("user_id = ?", userID).
		Count(&alertCount).Error
	if err != nil {
		return nil, fmt.Errorf("统计异动事件数量失败: %w", err)
	}

	// 获取未读异动数量
	var unreadAlertCount int64
	err = u.db.Model(&model.AlertEvent{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&unreadAlertCount).Error
	if err != nil {
		return nil, fmt.Errorf("统计未读异动数量失败: %w", err)
	}

	stats["subscription_count"] = subscriptionCount
	stats["alert_count"] = alertCount
	stats["unread_alert_count"] = unreadAlertCount

	return stats, nil
}

func (u *UserDB) GetLastLoginUsers(days int) ([]*model.User, error) {
	var users []*model.User
	cutoffTime := time.Now().AddDate(0, 0, -days)

	err := u.db.Where("last_login_at >= ?", cutoffTime).
		Order("last_login_at DESC").
		Find(&users).Error

	if err != nil {
		return nil, fmt.Errorf("查询最近登录用户失败: %w", err)
	}
	return users, nil
}

func (u *UserDB) GetInactiveUsers(days int) ([]*model.User, error) {
	var users []*model.User
	cutoffTime := time.Now().AddDate(0, 0, -days)

	err := u.db.Where("last_login_at < ? OR last_login_at IS NULL", cutoffTime).
		Find(&users).Error

	if err != nil {
		return nil, fmt.Errorf("查询不活跃用户失败: %w", err)
	}
	return users, nil
}

func (u *UserDB) GetTotalCount() (int64, error) {
	var count int64
	err := u.db.Model(&model.User{}).Count(&count).Error
	return count, err
}

func (u *UserDB) GetActiveCount() (int64, error) {
	var count int64
	err := u.db.Model(&model.User{}).Where("status = ?", 1).Count(&count).Error
	return count, err
}
