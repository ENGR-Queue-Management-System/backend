package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"src/helpers"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
)

func SendPushNotification(db *sql.DB, message string) error {
	rows, err := db.Query("SELECT student_id, endpoint, auth, p256dh FROM subscriptions")
	if err != nil {
		return fmt.Errorf("error querying subscriptions: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var endpoint, auth, p256dh string
		if err := rows.Scan(&endpoint, &auth, &p256dh); err != nil {
			return fmt.Errorf("error scanning row: %v", err)
		}

		sub := &webpush.Subscription{
			Endpoint: endpoint,
			Keys: webpush.Keys{
				Auth:   auth,
				P256dh: p256dh,
			},
		}

		payload := []byte(message)
		_, err := webpush.SendNotification(payload, sub, &webpush.Options{
			VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
			VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
		})

		if err != nil {
			log.Println("Error sending push notification:", err)
			return fmt.Errorf("error sending push notification: %v", err)
		}

		fmt.Printf("Sent notification to %s\n", endpoint)
	}

	return nil
}

func SendNotificationTrigger(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		message := `{"title": "Hello", "body": "This is a test notification"}`
		if err := SendPushNotification(db, message); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(map[string]string{"status": "notification sent"}))
	}
}
