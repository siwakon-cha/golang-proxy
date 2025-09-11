package database

import (
	"log"

	"rpc-proxy/internal/models"
)

func (db *GormDB) AutoMigrate() error {
	log.Println("Running GORM auto-migrations...")
	
	if err := models.AutoMigrate(db.DB); err != nil {
		return err
	}

	log.Println("Auto-migrations completed successfully")
	return nil
}

func (db *GormDB) SeedData() error {
	log.Println("Seeding default data...")
	
	if err := models.SeedDefaultData(db.DB); err != nil {
		return err
	}

	log.Println("Default data seeded successfully")
	return nil
}