package api

import (
	"database/sql"
	"net/http"
	"src/helpers"
	"src/models"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func SaveSubscription(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}
		claims := token.Claims.(jwt.MapClaims)
		firstName, ok := claims["firstName"].(string)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid firstName in token"})
			return
		}
		lastName, ok := claims["lastName"].(string)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lastName in token"})
			return
		}

		subscription := new(struct {
			Endpoint string `json:"endpoint"`
			Keys     struct {
				Auth   string `json:"auth"`
				P256dh string `json:"p256dh"`
			}
		})
		if err := c.Bind(&subscription); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var savedSubscription models.Subscription
		query := `INSERT INTO subscriptions (firstName, lastName, endpoint, auth, p256dh) VALUES ($1, $2, $3, $4, $5)
							ON CONFLICT (firstName, lastName, endpoint) DO UPDATE
							SET auth = EXCLUDED.auth, p256dh = EXCLUDED.p256dh
							RETURNING *`
		err = db.QueryRow(query, firstName, lastName, subscription.Endpoint, subscription.Keys.Auth, subscription.Keys.P256dh).Scan(
			&savedSubscription.FirstName,
			&savedSubscription.LastName,
			&savedSubscription.Endpoint,
			&savedSubscription.Auth,
			&savedSubscription.P256dh,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving subscription: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(savedSubscription))
	}
}

func RegisterRoutes(r *gin.RouterGroup, db *sql.DB) {
	r.POST("/subscribe", SaveSubscription(db))
	r.POST("/send-notification", SendNotificationTrigger(db))

	r.POST("/authentication", Authentication(db))
	r.POST("/reserve", ReserveNotLogin(db))

	r.GET("/user", GetUserInfo(db))

	r.GET("/counter", GetCounters(db))
	r.POST("/counter", CreateCounter(db))
	r.PUT("/counter/:id", UpdateCounter(db))
	r.DELETE("/counter/:id", DeleteCounter(db))

	r.GET("/topic", GetTopics(db))
	r.POST("/topic", CreateTopic(db))
	r.PUT("/topic/:id", UpdateTopic(db))
	r.DELETE("/topic/:id", DeleteTopic(db))
}
