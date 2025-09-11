package gorm

import (
	"fmt"

	"rpc-proxy/internal/database"
	"rpc-proxy/internal/models"
	"rpc-proxy/internal/repository"
	"rpc-proxy/internal/types"

	"gorm.io/gorm"
)

type rpcEndpointRepository struct {
	db *database.GormDB
}

func NewRPCEndpointRepository(db *database.GormDB) repository.RPCEndpointRepository {
	return &rpcEndpointRepository{db: db}
}

func (r *rpcEndpointRepository) GetAll() ([]*types.RPCEndpoint, error) {
	var endpoints []models.RPCEndpoint
	if err := r.db.Order("created_at ASC").Find(&endpoints).Error; err != nil {
		return nil, fmt.Errorf("failed to get all endpoints: %w", err)
	}

	return r.modelsToTypes(endpoints), nil
}

func (r *rpcEndpointRepository) GetEnabled() ([]*types.RPCEndpoint, error) {
	var endpoints []models.RPCEndpoint
	if err := r.db.Where("enabled = ?", true).Order("weight DESC, created_at ASC").Find(&endpoints).Error; err != nil {
		return nil, fmt.Errorf("failed to get enabled endpoints: %w", err)
	}

	return r.modelsToTypes(endpoints), nil
}

// GetEnabledByChain returns all enabled RPC endpoints for a specific chain
func (r *rpcEndpointRepository) GetEnabledByChain(chainName string) ([]*types.RPCEndpoint, error) {
	var endpoints []models.RPCEndpoint
	query := `
		SELECT re.* 
		FROM rpc_endpoints re
		JOIN chains c ON re.chain_id = c.id
		WHERE re.enabled = true AND c.name = ?
		ORDER BY re.weight DESC, re.created_at ASC
	`
	
	if err := r.db.Raw(query, chainName).Scan(&endpoints).Error; err != nil {
		return nil, fmt.Errorf("failed to get enabled endpoints for chain %s: %w", chainName, err)
	}
	
	result := r.modelsToTypes(endpoints)
	// Set chain name for each endpoint
	for _, endpoint := range result {
		endpoint.ChainName = chainName
	}
	
	return result, nil
}

// GetAllByChain returns all RPC endpoints for a specific chain
func (r *rpcEndpointRepository) GetAllByChain(chainName string) ([]*types.RPCEndpoint, error) {
	var endpoints []models.RPCEndpoint
	query := `
		SELECT re.*
		FROM rpc_endpoints re
		JOIN chains c ON re.chain_id = c.id
		WHERE c.name = ?
		ORDER BY re.weight DESC, re.created_at ASC
	`
	
	if err := r.db.Raw(query, chainName).Scan(&endpoints).Error; err != nil {
		return nil, fmt.Errorf("failed to get endpoints for chain %s: %w", chainName, err)
	}
	
	result := r.modelsToTypes(endpoints)
	// Set chain name for each endpoint
	for _, endpoint := range result {
		endpoint.ChainName = chainName
	}
	
	return result, nil
}

func (r *rpcEndpointRepository) GetByID(id int) (*types.RPCEndpoint, error) {
	var endpoint models.RPCEndpoint
	if err := r.db.First(&endpoint, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("endpoint with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to get endpoint by ID: %w", err)
	}

	return r.modelToType(&endpoint), nil
}

func (r *rpcEndpointRepository) GetByName(name string) (*types.RPCEndpoint, error) {
	var endpoint models.RPCEndpoint
	if err := r.db.Where("name = ?", name).First(&endpoint).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("endpoint with name %s not found", name)
		}
		return nil, fmt.Errorf("failed to get endpoint by name: %w", err)
	}

	return r.modelToType(&endpoint), nil
}

func (r *rpcEndpointRepository) Create(req *repository.CreateRPCEndpointRequest) (*types.RPCEndpoint, error) {
	endpoint := models.RPCEndpoint{
		Name:    req.Name,
		URL:     req.URL,
		Weight:  req.Weight,
		Enabled: req.Enabled,
	}

	if err := r.db.Create(&endpoint).Error; err != nil {
		return nil, fmt.Errorf("failed to create endpoint: %w", err)
	}

	return r.modelToType(&endpoint), nil
}

func (r *rpcEndpointRepository) Update(id int, req *repository.UpdateRPCEndpointRequest) (*types.RPCEndpoint, error) {
	var endpoint models.RPCEndpoint
	if err := r.db.First(&endpoint, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("endpoint with ID %d not found", id)
		}
		return nil, fmt.Errorf("failed to find endpoint: %w", err)
	}

	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.URL != nil {
		updates["url"] = *req.URL
	}
	if req.Weight != nil {
		updates["weight"] = *req.Weight
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	if len(updates) == 0 {
		return r.modelToType(&endpoint), nil
	}

	if err := r.db.Model(&endpoint).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update endpoint: %w", err)
	}

	// Reload the updated endpoint
	if err := r.db.First(&endpoint, id).Error; err != nil {
		return nil, fmt.Errorf("failed to reload updated endpoint: %w", err)
	}

	return r.modelToType(&endpoint), nil
}

func (r *rpcEndpointRepository) Delete(id int) error {
	result := r.db.Delete(&models.RPCEndpoint{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete endpoint: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("endpoint with ID %d not found", id)
	}

	return nil
}

func (r *rpcEndpointRepository) SetEnabled(id int, enabled bool) error {
	result := r.db.Model(&models.RPCEndpoint{}).Where("id = ?", id).Update("enabled", enabled)
	if result.Error != nil {
		return fmt.Errorf("failed to set endpoint enabled status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("endpoint with ID %d not found", id)
	}

	return nil
}

func (r *rpcEndpointRepository) UpdateHealthStatus(id int, healthy bool, responseTime int64, blockNumber string, errorMsg string) error {
	// Create health check record
	healthCheck := models.HealthCheck{
		EndpointID:     uint(id),
		Healthy:        healthy,
		ResponseTimeMs: responseTime,
		BlockNumber:    blockNumber,
		ErrorMessage:   errorMsg,
	}

	if err := r.db.Create(&healthCheck).Error; err != nil {
		return fmt.Errorf("failed to create health check: %w", err)
	}

	return nil
}

// Helper methods to convert between models and types
func (r *rpcEndpointRepository) modelToType(model *models.RPCEndpoint) *types.RPCEndpoint {
	return &types.RPCEndpoint{
		ID:           int(model.ID),
		Name:         model.Name,
		URL:          model.URL,
		Weight:       model.Weight,
		Enabled:      model.Enabled,
		ChainID:      int(model.ChainID),
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
		Healthy:      model.Healthy,
		LastCheck:    model.LastCheck,
		ResponseTime: model.ResponseTime,
		BlockNumber:  model.BlockNumber,
		FailCount:    model.FailCount,
	}
}

func (r *rpcEndpointRepository) modelsToTypes(models []models.RPCEndpoint) []*types.RPCEndpoint {
	types := make([]*types.RPCEndpoint, len(models))
	for i, model := range models {
		types[i] = r.modelToType(&model)
	}
	return types
}