package gorm

import (
	"fmt"
	"time"

	"rpc-proxy/internal/database"
	"rpc-proxy/internal/models"
	"rpc-proxy/internal/repository"
)

type healthCheckRepository struct {
	db *database.GormDB
}

func NewHealthCheckRepository(db *database.GormDB) repository.HealthCheckRepository {
	return &healthCheckRepository{db: db}
}

func (r *healthCheckRepository) Create(req *repository.CreateHealthCheckRequest) error {
	healthCheck := models.HealthCheck{
		EndpointID:     uint(req.EndpointID),
		Healthy:        req.Healthy,
		ResponseTimeMs: req.ResponseTimeMs,
		BlockNumber:    req.BlockNumber,
		ErrorMessage:   req.ErrorMessage,
	}

	if err := r.db.Create(&healthCheck).Error; err != nil {
		return fmt.Errorf("failed to create health check: %w", err)
	}

	return nil
}

func (r *healthCheckRepository) GetByEndpointID(endpointID int, limit int) ([]*repository.HealthCheck, error) {
	var healthChecks []models.HealthCheck
	if err := r.db.Where("endpoint_id = ?", endpointID).
		Order("checked_at DESC").
		Limit(limit).
		Find(&healthChecks).Error; err != nil {
		return nil, fmt.Errorf("failed to get health checks: %w", err)
	}

	return r.modelsToRepo(healthChecks), nil
}

func (r *healthCheckRepository) GetLatestByEndpointID(endpointID int) (*repository.HealthCheck, error) {
	var healthCheck models.HealthCheck
	if err := r.db.Where("endpoint_id = ?", endpointID).
		Order("checked_at DESC").
		First(&healthCheck).Error; err != nil {
		return nil, fmt.Errorf("failed to get latest health check: %w", err)
	}

	return r.modelToRepo(&healthCheck), nil
}

func (r *healthCheckRepository) DeleteOldRecords(days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days)
	
	result := r.db.Where("checked_at < ?", cutoffDate).Delete(&models.HealthCheck{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete old health check records: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		fmt.Printf("Deleted %d old health check records\n", result.RowsAffected)
	}

	return nil
}

// Helper methods to convert between models and repository types
func (r *healthCheckRepository) modelToRepo(model *models.HealthCheck) *repository.HealthCheck {
	return &repository.HealthCheck{
		ID:             int(model.ID),
		EndpointID:     int(model.EndpointID),
		Healthy:        model.Healthy,
		ResponseTimeMs: model.ResponseTimeMs,
		BlockNumber:    model.BlockNumber,
		ErrorMessage:   model.ErrorMessage,
		CheckedAt:      model.CheckedAt.Format(time.RFC3339),
	}
}

func (r *healthCheckRepository) modelsToRepo(models []models.HealthCheck) []*repository.HealthCheck {
	results := make([]*repository.HealthCheck, len(models))
	for i, model := range models {
		results[i] = r.modelToRepo(&model)
	}
	return results
}