package api

import (
	"database/sql"
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
)

func GetConfig(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `SELECT * FROM config LIMIT 1`
		var config models.Config
		err := dbConn.QueryRow(query).Scan(&config.ID, &config.LoginNotCmu)
		if err != nil {
			if err == sql.ErrNoRows {
				helpers.FormatErrorResponse(c, http.StatusNotFound, "Config not found")
			} else {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve config")
			}
			return
		}
		helpers.FormatSuccessResponse(c, config)
	}
}

func SetLoginNotCmu(dbConn *sql.DB, server *socketio.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := new(struct {
			LoginNotCmu bool `json:"loginNotCmu"`
		})
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		query := `UPDATE config SET login_not_cmu = $1`
		_, err := dbConn.Exec(query, body.LoginNotCmu)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update config")
			return
		}

		// server.BroadcastToNamespace(helpers.SOCKET, "setLoginNotCmu", body.LoginNotCmu)

		helpers.FormatSuccessResponse(c, map[string]interface{}{"message": "Config updated successfully"})
	}
}
