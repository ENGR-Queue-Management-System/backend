package api

import (
	"database/sql"
	"net/http"
	"src/helpers"
	"src/models"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

func SaveSubscription(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
		if err != nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
		}
		claims := token.Claims.(jwt.MapClaims)
		firstName, ok := claims["firstName"].(string)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid firstName in token")
		}
		lastName, ok := claims["lastName"].(string)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid lastName in token")
		}

		subscription := new(struct {
			Endpoint string `json:"endpoint"`
			Keys     struct {
				Auth   string `json:"auth"`
				P256dh string `json:"p256dh"`
			}
		})
		if err := c.Bind(&subscription); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
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
			return echo.NewHTTPError(http.StatusInternalServerError, "Error saving subscription: "+err.Error())
		}

		return c.JSON(http.StatusOK, helpers.FormatSuccessResponse(savedSubscription))
	}
}

func RegisterRoutes(e *echo.Group, db *sql.DB) {
	e.POST("/subscribe", SaveSubscription(db))
	e.POST("/send-notification", SendNotificationTrigger(db))

	e.POST("/authentication", Authentication(db))
	e.POST("/reserve", ReserveNotLogin(db))

	e.GET("/user", GetUserInfo(db))

	e.GET("/counter", GetCounters(db))
	e.POST("/counter", CreateCounter(db))
	e.PUT("/counter/:id", UpdateCounter(db))
	e.DELETE("/counter/:id", DeleteCounter(db))

	e.GET("/topic", GetTopics(db))
	e.POST("/topic", CreateTopic(db))
	e.PUT("/topic/:id", UpdateTopic(db))
	e.DELETE("/topic/:id", DeleteTopic(db))
}
