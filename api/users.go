package api

import (
	"database/sql"
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/labstack/echo/v4"
)

func GetUserInfo(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		email, err := helpers.ExtractEmailFromToken(c)
		if err != nil {
			return err
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
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}

		user.Counter = counter
		return c.JSON(http.StatusOK, helpers.FormatSuccessResponse(user))
	}
}
