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
		studentId, ok := claims["studentId"].(string)
		if !ok {
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid studentId in token")
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
		query := `INSERT INTO subscriptions (student_id, endpoint, auth, p256dh) VALUES ($1, $2, $3, $4)
							ON CONFLICT (student_id, endpoint) DO UPDATE
							SET auth = EXCLUDED.auth, p256dh = EXCLUDED.p256dh
							RETURNING *`
		err = db.QueryRow(query, studentId, subscription.Endpoint, subscription.Keys.Auth, subscription.Keys.P256dh).Scan(
			&savedSubscription.StudentID,
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
	e.GET("/user", GetUserInfo(db))
	e.GET("/counter", GetCounters(db))
	e.POST("/counter", CreateCounter(db))
	e.DELETE("/counter/:id", DeleteCounter(db))
}
