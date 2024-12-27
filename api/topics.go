package api

import (
	"database/sql"
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
)

func GetTopics(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `SELECT * FROM topics`
		rows, err := dbConn.Query(query)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch topics")
			return
		}
		defer rows.Close()

		var topics []models.Topic
		for rows.Next() {
			var topic models.Topic
			if err := rows.Scan(
				&topic.ID, &topic.TopicTH, &topic.TopicEN, &topic.Code,
			); err != nil {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to read topic data")
				return
			}
			topics = append(topics, topic)
		}

		if err := rows.Err(); err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Error iterating topics")
			return
		}
		helpers.FormatSuccessResponse(c, topics)
	}
}

func CreateTopic(dbConn *sql.DB, server *socketio.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		helpers.FormatSuccessResponse(c, map[string]string{
			"message": "not create api",
		})
	}
}

func UpdateTopic(dbConn *sql.DB, server *socketio.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		helpers.FormatSuccessResponse(c, map[string]string{
			"message": "not create api",
		})
	}
}

func DeleteTopic(dbConn *sql.DB, server *socketio.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := dbConn.Exec("DELETE FROM topics WHERE id = $1", id)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to delete topic")
			return
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to verify deletion")
			return
		}
		if rowsAffected == 0 {
			helpers.FormatErrorResponse(c, http.StatusNotFound, "Topic not found")
			return
		}
		helpers.FormatSuccessResponse(c, map[string]string{"message": "Topic deleted successfully"})
	}
}
