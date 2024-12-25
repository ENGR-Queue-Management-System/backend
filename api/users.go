package api

import (
	"database/sql"
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
)

func GetUserInfo(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := helpers.ExtractToken(c)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusUnauthorized, err.Error())
			return
		}
		email, ok := (*claims)["email"].(string)
		if !ok || email == "" {
			helpers.FormatErrorResponse(c, http.StatusUnauthorized, "Email claim is missing or invalid in token")
			return
		}
		query := `SELECT * FROM users u
		LEFT JOIN counters c
		ON u.counter_id = c.id
		WHERE email = $1`
		row := dbConn.QueryRow(query, email)
		var user models.User
		var counter models.Counter
		err = row.Scan(
			&user.ID, &user.FirstNameTH, &user.LastNameTH,
			&user.FirstNameEN, &user.LastNameEN, &user.Email, &user.CounterID,
			&counter.ID, &counter.Counter, &counter.Status, &counter.TimeClosed,
		)
		if err == sql.ErrNoRows {
			helpers.FormatErrorResponse(c, http.StatusNotFound, "User not found")
			return
		}

		user.Counter = counter
		helpers.FormatSuccessResponse(c, user)
	}
}
