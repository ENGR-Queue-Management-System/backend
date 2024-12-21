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

func SendPushNotification(db *sql.DB, message string, userIdentifier map[string]string) error {
	query := "SELECT endpoint, auth, p256dh FROM subscriptions WHERE firstname = $1 AND lastname = $2"
	rows, err := db.Query(query, userIdentifier["firstName"], userIdentifier["lastName"])
	if err != nil {
		return fmt.Errorf("error querying subscriptions: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var endpoint, auth, p256dh string
		if err := rows.Scan(&endpoint, &auth, &p256dh); err != nil {
			return fmt.Errorf("error scanning row: %v", err)
		}

		options := &webpush.Options{
			VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
			VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
			TTL:             60,
		}
		response, err := webpush.SendNotification([]byte(message), &webpush.Subscription{
			Endpoint: endpoint,
			Keys: webpush.Keys{
				Auth:   auth,
				P256dh: p256dh,
			},
		}, options)
		if err != nil {
			log.Printf("Error sending notification to %s: %v", endpoint, err)
		} else {
			log.Printf("Successfully sent notification to %s. Response status: %s", endpoint, response.Status)
		}

		fmt.Printf("Sent notification to %s\n", endpoint)
	}

	return nil
}

func SendNotificationTrigger(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := new(struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			Message   string `json:"message"`
		})
		if err := c.Bind(body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		if err := SendPushNotification(db, body.Message, map[string]string{
			"firstName": body.FirstName,
			"lastName":  body.LastName,
		}); err != nil {
			log.Printf("Error sending notification: %v", err) // More detailed logging
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(map[string]string{"status": "notification sent"}))
	}
}

func GetSubscription(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := "SELECT firstname, lastname FROM subscriptions"
		rows, err := db.Query(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error executing query: %v", err)})
			return
		}
		defer rows.Close()

		var subscriptions []map[string]string
		for rows.Next() {
			var firstName, lastName string
			if err := rows.Scan(&firstName, &lastName); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error scanning row: %v", err)})
				return
			}
			subscription := map[string]string{
				"firstName": firstName,
				"lastName":  lastName,
			}
			subscriptions = append(subscriptions, subscription)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("error iterating over rows: %v", err)})
			return
		}
		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(subscriptions))
	}
}
