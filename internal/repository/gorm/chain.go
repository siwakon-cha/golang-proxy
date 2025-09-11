package gorm

import (
	"fmt"

	"rpc-proxy/internal/database"
	"rpc-proxy/internal/models"
	"rpc-proxy/internal/types"
)

type ChainRepository struct {
	db *database.GormDB
}

func NewChainRepository(db *database.GormDB) *ChainRepository {
	return &ChainRepository{db: db}
}

func (r *ChainRepository) GetAll() ([]*types.Chain, error) {
	var chains []*models.Chain
	if err := r.db.DB.Where("is_enabled = ?", true).Find(&chains).Error; err != nil {
		return nil, fmt.Errorf("failed to get all chains: %w", err)
	}

	result := make([]*types.Chain, len(chains))
	for i, chain := range chains {
		result[i] = r.modelToType(chain)
	}

	return result, nil
}

func (r *ChainRepository) GetByName(name string) (*types.Chain, error) {
	var chain models.Chain
	if err := r.db.DB.Where("name = ? AND is_enabled = ?", name, true).First(&chain).Error; err != nil {
		return nil, fmt.Errorf("failed to get chain by name %s: %w", name, err)
	}

	return r.modelToType(&chain), nil
}

func (r *ChainRepository) GetByChainID(chainID int) (*types.Chain, error) {
	var chain models.Chain
	if err := r.db.DB.Where("chain_id = ? AND is_enabled = ?", chainID, true).First(&chain).Error; err != nil {
		return nil, fmt.Errorf("failed to get chain by chain_id %d: %w", chainID, err)
	}

	return r.modelToType(&chain), nil
}

func (r *ChainRepository) GetByRPCPath(rpcPath string) (*types.Chain, error) {
	var chain models.Chain
	if err := r.db.DB.Where("rpc_path = ? AND is_enabled = ?", rpcPath, true).First(&chain).Error; err != nil {
		return nil, fmt.Errorf("failed to get chain by rpc_path %s: %w", rpcPath, err)
	}

	return r.modelToType(&chain), nil
}

func (r *ChainRepository) Create(chain *types.Chain) error {
	model := r.typeToModel(chain)
	if err := r.db.DB.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create chain: %w", err)
	}

	chain.ID = int(model.ID)
	return nil
}

func (r *ChainRepository) Update(chain *types.Chain) error {
	model := r.typeToModel(chain)
	if err := r.db.DB.Save(model).Error; err != nil {
		return fmt.Errorf("failed to update chain: %w", err)
	}

	return nil
}

func (r *ChainRepository) Delete(id int) error {
	if err := r.db.DB.Delete(&models.Chain{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete chain with id %d: %w", id, err)
	}

	return nil
}

func (r *ChainRepository) modelToType(m *models.Chain) *types.Chain {
	if m == nil {
		return nil
	}

	return &types.Chain{
		ID:                   int(m.ID),
		ChainID:              m.ChainID,
		Name:                 m.Name,
		DisplayName:          m.DisplayName,
		RPCPath:              m.RPCPath,
		IsTestnet:            m.IsTestnet,
		IsEnabled:            m.IsEnabled,
		NativeCurrencySymbol: m.NativeCurrencySymbol,
		BlockExplorerURL:     m.BlockExplorerURL,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
	}
}

func (r *ChainRepository) typeToModel(t *types.Chain) *models.Chain {
	if t == nil {
		return nil
	}

	return &models.Chain{
		ID:                   uint(t.ID),
		ChainID:              t.ChainID,
		Name:                 t.Name,
		DisplayName:          t.DisplayName,
		RPCPath:              t.RPCPath,
		IsTestnet:            t.IsTestnet,
		IsEnabled:            t.IsEnabled,
		NativeCurrencySymbol: t.NativeCurrencySymbol,
		BlockExplorerURL:     t.BlockExplorerURL,
		CreatedAt:            t.CreatedAt,
		UpdatedAt:            t.UpdatedAt,
	}
}