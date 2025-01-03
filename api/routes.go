package api

import (
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
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

func RegisterRoutes(r *gin.RouterGroup, db *gorm.DB, server *socketio.Server) {
	r.POST("/subscribe", SaveSubscription(db))
	r.POST("/send-notification", SendNotificationTrigger(db))
	r.GET("/test-send-noti", GetSubscription(db))

	r.POST("/authentication", Authentication(db))

	r.GET("/config", GetConfig(db))
	r.PUT("/config/login-not-cmu", SetLoginNotCmu(db, server))

	r.GET("/user", GetUserInfo(db))

	r.GET("/counter", GetCounters(db))
	r.POST("/counter", CreateCounter(db, server))
	r.PUT("/counter/:id", UpdateCounter(db, server))
	r.DELETE("/counter/:id", DeleteCounter(db, server))

	r.GET("/topic", GetTopics(db))
	r.POST("/topic", CreateTopic(db, server))
	r.PUT("/topic/:id", UpdateTopic(db, server))
	r.DELETE("/topic/:id", DeleteTopic(db, server))

	r.GET("/queue", GetQueues(db))
	r.GET("/queue/student", GetStudentQueue(db))
	r.POST("/queue", CreateQueue(db, server))
	r.PUT("/queue/:id", UpdateQueue(db, server))
	r.DELETE("/queue/:id", DeleteQueue(db, server))
}
