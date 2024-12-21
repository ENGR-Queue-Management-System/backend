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
		email, err := helpers.ExtractEmailFromToken(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		user.Counter = counter
		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(user))
	}
}
