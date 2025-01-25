package api

import (
	"net/http"
	"src/helpers"
	"src/middleware"
	"src/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func SaveSubscription(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userClaims, ok := helpers.ExtractClaims(c)
		if !ok {
			return
		}
		firstName, ok := userClaims["firstName"].(string)
		if !ok {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid firstName in token")
			return
		}
		lastName, ok := userClaims["lastName"].(string)
		if !ok {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid lastName in token")
			return
		}

		var body struct {
			Platform string `json:"platform"`
			Endpoint string `json:"endpoint"`
			Keys     struct {
				Auth   string `json:"auth"`
				P256dh string `json:"p256dh"`
			} `json:"keys"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid JSON payload: "+err.Error())
			return
		}

		subscription := models.Subscription{
			FirstName: firstName,
			LastName:  lastName,
			Platform:  body.Platform,
			Endpoint:  body.Endpoint,
			Auth:      body.Keys.Auth,
			P256dh:    body.Keys.P256dh,
		}
		err := db.Clauses(
			clause.OnConflict{
				Columns:   []clause.Column{{Name: "first_name"}, {Name: "last_name"}, {Name: "platform"}},
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

	r.POST("/authentication", Authentication(db))
	r.GET("/config", GetConfig(db))

	r.GET("/counter", GetCounters(db))
	r.GET("/topic", GetTopics(db))
	r.POST("/queue", CreateQueue(db, hub))

	protected := r.Group("/")
	protected.Use(middleware.AuthRequired())
	{
		protected.POST("/subscribe", SaveSubscription(db))
		protected.POST("/send-notification", SendNotificationTrigger(db, hub))

		protected.GET("/user", GetUserInfo(db))

		protected.PUT("/config/login-not-cmu", SetLoginNotCmu(db, hub))
		protected.PUT("/config/audio", SetAudio(db, hub))

		protected.POST("/counter", CreateCounter(db, hub))
		protected.PUT("/counter/:id", UpdateCounter(db, hub))
		protected.DELETE("/counter/:id", DeleteCounter(db, hub))

		protected.POST("/topic", CreateTopic(db, hub))
		protected.PUT("/topic/:id", UpdateTopic(db, hub))
		protected.DELETE("/topic/:id", DeleteTopic(db, hub))

		protected.GET("/queue", GetQueues(db))
		protected.GET("/queue/student", GetStudentQueue(db))
		protected.GET("/queue/called", GetCalledQueues(db))
		protected.PUT("/queue/feedback/:id", UpdateQueueFeedback(db))
		protected.PUT("/queue/:id", UpdateQueue(db, hub))
		protected.DELETE("/queue/:id", DeleteQueue(db, hub))

		protected.GET("/feedback", GetFeedbackByUser(db))
		protected.POST("/feedback", CreateFeedback(db))

		protected.GET("/noti-schedule", GetNotiSchedule(db))
		protected.POST("/noti-schedule", CreateNotiSchedule(db))
	}
}
