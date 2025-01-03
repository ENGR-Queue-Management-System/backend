package db

import (
	"fmt"
	"log"
	"src/models"
	"time"

	"gorm.io/gorm"
)

func StartCounterStatusUpdater(db *gorm.DB, interval time.Duration) {
	go func() {
		for {
			err := UpdateCounterStatus(db)
			if err != nil {
				log.Printf("Error updating counter status: %v", err)
			}
			time.Sleep(interval)
		}
	}()
}

func UpdateCounterStatus(db *gorm.DB) error {
	now := time.Now()
	startTime := now.Add(-1 * time.Minute)
	endTime := now.Add(1 * time.Minute)

	result := db.Model(&models.Counter{}).
		Where("time_closed BETWEEN ? AND ? AND status = ?", startTime, endTime, true).
		Update("status", false)

	if result.Error != nil {
		return fmt.Errorf("failed to update counter status: %v", result.Error)
	}

	log.Printf("Successfully updated %d counters' status", result.RowsAffected)
	return nil
}
