package api

import (
	"database/sql"
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/labstack/echo/v4"
)

func GetTopics(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		query := `SELECT * FROM topics`
		rows, err := dbConn.Query(query)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch topics"})
		}
		defer rows.Close()

		var topics []models.Topic
		for rows.Next() {
			var topic models.Topic
			if err := rows.Scan(
				&topic.ID, &topic.TopicTH, &topic.TopicEN, &topic.Code,
			); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read topic data"})
			}
			topics = append(topics, topic)
		}

		if err := rows.Err(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error iterating topics"})
		}
		return c.JSON(http.StatusOK, helpers.FormatSuccessResponse(topics))
	}
}

func CreateTopic(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusCreated, map[string]string{
			"message": "not create api",
		})
	}
}

func UpdateTopic(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusCreated, map[string]string{
			"message": "not create api",
		})
	}
}

func DeleteTopic(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		result, err := dbConn.Exec("DELETE FROM topics WHERE id = $1", id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete topic"})
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to verify deletion"})
		}
		if rowsAffected == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Topic not found"})
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "Topic deleted successfully"})
	}
}
