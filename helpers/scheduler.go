package helpers

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

func StartCounterStatusUpdater(dbConn *sql.DB, interval time.Duration) {
	go func() {
		for {
			err := UpdateCounterStatus(dbConn)
			if err != nil {
				log.Printf("Error updating counter status: %v", err)
			}
			time.Sleep(interval)
		}
	}()
}

func UpdateCounterStatus(dbConn *sql.DB) error {
	query := `
		UPDATE counters
		SET status = false
		WHERE time_closed <= (NOW()::time + interval '1 minute')
			AND time_closed >= (NOW()::time - interval '1 minute')
			AND status = true;
	`
	_, err := dbConn.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to update counter status: %v", err)
	}
	return nil
}
