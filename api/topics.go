package api

import (
	"database/sql"
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
)

func GetTopics(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := `SELECT * FROM topics`
		rows, err := dbConn.Query(query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch topics"})
			return
		}
		defer rows.Close()

		var topics []models.Topic
		for rows.Next() {
			var topic models.Topic
			if err := rows.Scan(
				&topic.ID, &topic.TopicTH, &topic.TopicEN, &topic.Code,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read topic data"})
				return
			}
			topics = append(topics, topic)
		}

		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating topics"})
			return
		}
		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(topics))
	}
}

func CreateTopic(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, map[string]string{
			"message": "not create api",
		})
	}
}

func UpdateTopic(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, map[string]string{
			"message": "not create api",
		})
	}
}

func DeleteTopic(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := dbConn.Exec("DELETE FROM topics WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete topic"})
			return
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify deletion"})
			return
		}
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
			return
		}
		c.JSON(http.StatusOK, map[string]string{"message": "Topic deleted successfully"})
	}
}
