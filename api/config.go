package api

import (
	"database/sql"
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
)

func GetConfig(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `SELECT * FROM config LIMIT 1`
		var config models.Config
		err := dbConn.QueryRow(query).Scan(&config.ID, &config.LoginNotCmu)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Config not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve config"})
			}
			return
		}
		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(config))
	}
}

func SetLoginNotCmu(dbConn *sql.DB) gin.HandlerFunc {
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
			return
		}

		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(map[string]interface{}{"message": "Config updated successfully"}))
	}
}
