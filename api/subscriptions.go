package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"src/helpers"
	"src/models"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SendPushNotification(db *gorm.DB, message string, userIdentifier map[string]string) error {
	var subscriptions []models.Subscription
	err := db.Where("first_name = ? AND last_name = ?", userIdentifier["firstName"], userIdentifier["lastName"]).Find(&subscriptions).Error
	if err != nil {
		return fmt.Errorf("error fetching subscriptions: %v", err)
	}

	for _, subscription := range subscriptions {
		options := &webpush.Options{
			Subscriber:      "worapit2002@gmail.com",
			VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
			VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
			TTL:             60,
			Urgency:         "high",
		}

		response, err := webpush.SendNotification([]byte(message), &webpush.Subscription{
			Endpoint: subscription.Endpoint,
			Keys: webpush.Keys{
				Auth:   subscription.Auth,
				P256dh: subscription.P256dh,
			},
		}, options)

		if err != nil {
			log.Printf("Error sending notification to %s: %v", subscription.Endpoint, err)
		} else {
			log.Printf("Successfully sent notification to %s. Response status: %s", subscription.Endpoint, response.Status)
		}

		fmt.Printf("Sent notification to %s\n", subscription.Endpoint)
	}

	return nil
}

func SendNotificationTrigger(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := new(struct {
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			Message   string `json:"message"`
		})
		if err := c.Bind(body); err != nil {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := SendPushNotification(db, body.Message, map[string]string{
			"firstName": body.FirstName,
			"lastName":  body.LastName,
		}); err != nil {
			log.Printf("Error sending notification: %v", err)
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, err.Error())
			return
		}

		helpers.FormatSuccessResponse(c, map[string]string{"status": "notification sent"})
	}
}

func GetSubscription(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var subscriptions []models.Subscription
		err := db.Find(&subscriptions).Error
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("error retrieving subscriptions: %v", err))
			return
		}

		var subscriptionList []map[string]string
		for _, sub := range subscriptions {
			subscription := map[string]string{
				"firstName": sub.FirstName,
				"lastName":  sub.LastName,
			}
			subscriptionList = append(subscriptionList, subscription)
		}

		helpers.FormatSuccessResponse(c, subscriptionList)
	}
}
