package api

import (
	"database/sql"
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/labstack/echo/v4"
)

func FormatUserData() {

}

func GetUserInfo(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		email, err := helpers.ExtractEmailFromToken(c)
		if err != nil {
			return err
		}
		row := dbConn.QueryRow("SELECT * FROM users WHERE email = $1", email)
		var user models.User
		err = row.Scan(&user.ID, &user.Firstname, &user.Lastname, &user.Email, &user.RoomID)
		if err == sql.ErrNoRows {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		}
		return c.JSON(http.StatusOK, user)
	}
}

func UpdateUser(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		email, err := helpers.ExtractEmailFromToken(c)
		if err != nil {
			return err
		}
		body := new(struct {
			Room int `json:"room"`
		})
		if err := c.Bind(body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
		}
		_, err = dbConn.Exec("UPDATE users SET name = $1 WHERE email = $2", body.Room, email)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user"})
		}

		row := dbConn.QueryRow("SELECT * FROM users WHERE email = $1", email)

		var updatedUser models.User
		if err := row.Scan(&updatedUser.ID, &updatedUser.Firstname, &updatedUser.Lastname, &updatedUser.Email, &updatedUser.RoomID, &updatedUser.Room); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch updated user data"})
		}

		return c.JSON(http.StatusOK, updatedUser)
	}
}
