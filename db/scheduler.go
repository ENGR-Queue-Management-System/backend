package db

import (
	"encoding/json"
	"fmt"
	"log"
	"src/api"
	"src/helpers"
	"src/models"
	"time"

	"gorm.io/gorm"
)

func StartCounterStatusUpdater(db *gorm.DB, interval time.Duration, hub *api.Hub) {
	go func() {
		for {
			err := UpdateCounterStatus(db, hub)
			if err != nil {
				log.Printf("Error updating counter status: %v", err)
			}
			time.Sleep(interval)
		}
	}()
}

func UpdateCounterStatus(db *gorm.DB, hub *api.Hub) error {
	now := helpers.GetBangkokTime()
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
		var updatedCounterIDs []int
		if err := tx.Model(&models.Counter{}).
			Where("time_closed BETWEEN ? AND ? AND status = ?", startTime, endTime, false).
			Pluck("id", &updatedCounterIDs).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to fetch updated counters: %v", err)
		}

		message, _ := json.Marshal(map[string]interface{}{
			"event": "updateCounterStatus",
			"data":  updatedCounterIDs,
		})
		hub.Broadcast(message)

		var affectedQueue []models.Queue
		err := tx.Model(&models.Queue{}).
			Where("counter_id IN (?) AND status = ?", updatedCounterIDs, helpers.IN_PROGRESS).
			Find(&affectedQueue).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			tx.Rollback()
			return fmt.Errorf("failed to update queue status: %v", err)
		}
		result := tx.Model(&models.Queue{}).
			Where("id IN (?)", getQueueIDs(affectedQueue)).
			Updates(map[string]interface{}{"status": helpers.CALLED})
		if result.Error != nil {
			tx.Rollback()
			return fmt.Errorf("failed to update queue status: %v", result.Error)
		}
		for _, queue := range affectedQueue {
			message := map[string]interface{}{
				"title": map[string]string{
					"en": "Let's review your recent help!",
					"th": "มารีวิวการบริการที่คุณได้รับกันเถอะ!",
				},
				"body": map[string]string{
					"en": "Was the service okay? Tap here to review.",
					"th": "การให้บริการโอเคไหม? แตะที่นี่เพื่อให้คะแนนเลย!",
				},
			}
			userIdentifier := map[string]string{
				"firstName": queue.Firstname,
				"lastName":  queue.Lastname,
			}
			go func(queue models.Queue) {
				messageJSON, err := json.Marshal(message)
				if err != nil {
					log.Printf("Error creating notification message for queue %d: %v", queue.ID, err)
					return
				}
				err = api.SendPushNotification(db, hub, string(messageJSON), userIdentifier, nil)
				if err != nil {
					log.Printf("Error sending notification for queue %d: %v", queue.ID, err)
				}
			}(queue)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("Successfully updated %d counters' status", result.RowsAffected)
	return nil
}

func getQueueIDs(queues []models.Queue) []int {
	ids := make([]int, len(queues))
	for i, q := range queues {
		ids[i] = q.ID
	}
	return ids
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
	thresholdDate := helpers.GetBangkokTime().AddDate(0, 0, -30)

	result := db.Where("created_at < ?", thresholdDate).Delete(&models.Queue{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete old queue entries: %v", result.Error)
	}

	log.Printf("Successfully deleted %d old queue entries", result.RowsAffected)
	return nil
}
