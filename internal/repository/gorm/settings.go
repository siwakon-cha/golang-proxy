package gorm

import (
	"fmt"

	"rpc-proxy/internal/database"
	"rpc-proxy/internal/models"
	"rpc-proxy/internal/repository"

	"gorm.io/gorm"
)

type settingsRepository struct {
	db *database.GormDB
}

func NewSettingsRepository(db *database.GormDB) repository.SettingsRepository {
	return &settingsRepository{db: db}
}

func (r *settingsRepository) Get(key string) (string, error) {
	var setting models.Setting
	if err := r.db.Where("key = ?", key).First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("setting with key %s not found", key)
		}
		return "", fmt.Errorf("failed to get setting: %w", err)
	}

	return setting.Value, nil
}

func (r *settingsRepository) Set(key, value, description string) error {
	setting := models.Setting{
		Key:         key,
		Value:       value,
		Description: description,
	}

	// Use GORM's Save method which will update if exists or create if not
	if err := r.db.Save(&setting).Error; err != nil {
		return fmt.Errorf("failed to set setting: %w", err)
	}

	return nil
}

func (r *settingsRepository) GetAll() (map[string]string, error) {
	var settings []models.Setting
	if err := r.db.Order("key").Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("failed to get all settings: %w", err)
	}

	result := make(map[string]string)
	for _, setting := range settings {
		result[setting.Key] = setting.Value
	}

	return result, nil
}

func (r *settingsRepository) Delete(key string) error {
	result := r.db.Where("key = ?", key).Delete(&models.Setting{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete setting: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("setting with key %s not found", key)
	}

	return nil
}