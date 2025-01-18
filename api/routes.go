package api

import (
	"encoding/json"
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func SaveSubscription(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := helpers.ExtractToken(c)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusUnauthorized, err.Error())
			return
		}
		firstName, ok := (*claims)["firstName"].(string)
		if !ok {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid firstName in token")
			return
		}
		lastName, ok := (*claims)["lastName"].(string)
		if !ok {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid lastName in token")
			return
		}

		var subscriptionPayload struct {
			Endpoint string `json:"endpoint"`
			Keys     struct {
				Auth   string `json:"auth"`
				P256dh string `json:"p256dh"`
			} `json:"keys"`
		}
		if err := c.ShouldBindJSON(&subscriptionPayload); err != nil {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
			return
		}

		subscription := models.Subscription{
			FirstName: firstName,
			LastName:  lastName,
			Endpoint:  subscriptionPayload.Endpoint,
			Auth:      subscriptionPayload.Keys.Auth,
			P256dh:    subscriptionPayload.Keys.P256dh,
		}
		err = db.Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: "first_name"}, {Name: "last_name"}},
				DoUpdates: clause.AssignmentColumns([]string{"endpoint", "auth", "p256dh"}),
			},
		).Create(&subscription).Error
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Error saving subscription: "+err.Error())
			return
		}

		helpers.FormatSuccessResponse(c, subscription)
	}
}

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, hub *Hub) {
	r.GET("/socket", func(c *gin.Context) {
		message, _ := json.Marshal(map[string]interface{}{
			"event": "trigger",
		})
		hub.broadcast <- message
		helpers.FormatSuccessResponse(c, map[string]string{"message": "success"})
	})

	r.POST("/subscribe", SaveSubscription(db))
	r.POST("/send-notification", SendNotificationTrigger(db, hub))

	r.POST("/authentication", Authentication(db))

	r.GET("/config", GetConfig(db))
	r.PUT("/config/login-not-cmu", SetLoginNotCmu(db, hub))

	r.GET("/user", GetUserInfo(db))

	r.GET("/counter", GetCounters(db))
	r.POST("/counter", CreateCounter(db, hub))
	r.PUT("/counter/:id", UpdateCounter(db, hub))
	r.DELETE("/counter/:id", DeleteCounter(db, hub))

	r.GET("/topic", GetTopics(db))
	r.POST("/topic", CreateTopic(db, hub))
	r.PUT("/topic/:id", UpdateTopic(db, hub))
	r.DELETE("/topic/:id", DeleteTopic(db, hub))

	r.GET("/queue", GetQueues(db))
	r.GET("/queue/student", GetStudentQueue(db))
	r.GET("/queue/called", GetCalledQueues(db))
	r.PUT("/queue/feedback/:id", UpdateQueueFeedback(db))
	r.POST("/queue", CreateQueue(db, hub))
	r.PUT("/queue/:id", UpdateQueue(db, hub))
	r.DELETE("/queue/:id", DeleteQueue(db, hub))

	r.GET("/feedback", GetFeedbackByUser(db))
	r.POST("/feedback", CreateFeedback(db))
}
