package gorm

import (
	"fmt"

	"rpc-proxy/internal/database"
	"rpc-proxy/internal/models"
	"rpc-proxy/internal/types"
)

type ChainConfigRepository struct {
	db *database.GormDB
}

func NewChainConfigRepository(db *database.GormDB) *ChainConfigRepository {
	return &ChainConfigRepository{db: db}
}

func (r *ChainConfigRepository) GetByChainID(chainID int) (map[string]string, error) {
	var configs []*models.ChainConfig
	if err := r.db.DB.Where("chain_id = ?", chainID).Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to get chain configs for chain_id %d: %w", chainID, err)
	}

	result := make(map[string]string)
	for _, config := range configs {
		result[config.ConfigKey] = config.ConfigValue
	}

	return result, nil
}

func (r *ChainConfigRepository) GetByChainName(chainName string) (map[string]string, error) {
	var configs []*models.ChainConfig
	if err := r.db.DB.
		Joins("JOIN chains ON chains.id = chain_configs.chain_id").
		Where("chains.name = ?", chainName).
		Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to get chain configs for chain %s: %w", chainName, err)
	}

	result := make(map[string]string)
	for _, config := range configs {
		result[config.ConfigKey] = config.ConfigValue
	}

	return result, nil
}

func (r *ChainConfigRepository) GetConfig(chainID int, configKey string) (string, error) {
	var config models.ChainConfig
	if err := r.db.DB.Where("chain_id = ? AND config_key = ?", chainID, configKey).First(&config).Error; err != nil {
		return "", fmt.Errorf("failed to get config %s for chain_id %d: %w", configKey, chainID, err)
	}

	return config.ConfigValue, nil
}

func (r *ChainConfigRepository) GetAll() ([]*types.ChainConfig, error) {
	var configs []*models.ChainConfig
	if err := r.db.DB.Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to get all chain configs: %w", err)
	}

	result := make([]*types.ChainConfig, len(configs))
	for i, config := range configs {
		result[i] = r.modelToType(config)
	}

	return result, nil
}

func (r *ChainConfigRepository) SetConfig(chainID int, configKey, configValue, description string) error {
	config := &models.ChainConfig{
		ChainID:     uint(chainID),
		ConfigKey:   configKey,
		ConfigValue: configValue,
		Description: description,
	}

	if err := r.db.DB.Save(config).Error; err != nil {
		return fmt.Errorf("failed to set config %s for chain_id %d: %w", configKey, chainID, err)
	}

	return nil
}

func (r *ChainConfigRepository) DeleteConfig(chainID int, configKey string) error {
	if err := r.db.DB.Where("chain_id = ? AND config_key = ?", chainID, configKey).Delete(&models.ChainConfig{}).Error; err != nil {
		return fmt.Errorf("failed to delete config %s for chain_id %d: %w", configKey, chainID, err)
	}

	return nil
}

func (r *ChainConfigRepository) DeleteAllByChainID(chainID int) error {
	if err := r.db.DB.Where("chain_id = ?", chainID).Delete(&models.ChainConfig{}).Error; err != nil {
		return fmt.Errorf("failed to delete all configs for chain_id %d: %w", chainID, err)
	}

	return nil
}

func (r *ChainConfigRepository) modelToType(m *models.ChainConfig) *types.ChainConfig {
	if m == nil {
		return nil
	}

	return &types.ChainConfig{
		ID:          int(m.ID),
		ChainID:     int(m.ChainID),
		ConfigKey:   m.ConfigKey,
		ConfigValue: m.ConfigValue,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func (r *ChainConfigRepository) typeToModel(t *types.ChainConfig) *models.ChainConfig {
	if t == nil {
		return nil
	}

	return &models.ChainConfig{
		ID:          uint(t.ID),
		ChainID:     uint(t.ChainID),
		ConfigKey:   t.ConfigKey,
		ConfigValue: t.ConfigValue,
		Description: t.Description,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}