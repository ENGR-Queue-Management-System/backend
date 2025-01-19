package db

import (
	"fmt"
	"log"
	"src/helpers"
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

	tx := db.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %v", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	result := tx.Model(&models.Counter{}).
		Where("time_closed BETWEEN ? AND ? AND status = ?", startTime, endTime, true).
		Update("status", false)

	if result.Error != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update counter status: %v", result.Error)
	}

	if result.RowsAffected > 0 {
		err := tx.Model(&models.Queue{}).
			Where("counter_id IN (?) AND status = ?", tx.Model(&models.Counter{}).Select("id").Where("status = false"), helpers.IN_PROGRESS).
			Update("status", helpers.CALLED).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return fmt.Errorf("failed to update queue status: %v", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("Successfully updated %d counters' status", result.RowsAffected)
	return nil
}

func StartQueueCleanup(db *gorm.DB, interval time.Duration) {
	go func() {
		for {
			err := DeleteOldQueueEntries(db)
			if err != nil {
				log.Printf("Error deleting old queue entries: %v", err)
			}
			time.Sleep(interval)
		}
	}()
}

func DeleteOldQueueEntries(db *gorm.DB) error {
	thresholdDate := time.Now().AddDate(0, 0, -30)

	result := db.Where("created_at < ?", thresholdDate).Delete(&models.Queue{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete old queue entries: %v", result.Error)
	}

	log.Printf("Successfully deleted %d old queue entries", result.RowsAffected)
	return nil
}
