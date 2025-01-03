package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"github.com/lib/pq"
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
		var body struct {
			TopicTH string `json:"topicTH"`
			TopicEN string `json:"topicEN"`
			Code    string `json:"code"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid request body")
			return
		}

		query := `INSERT INTO topics (topic_th, topic_en, code) VALUES ($1, $2, $3) RETURNING id`
		var newTopicID int
		err := dbConn.QueryRow(query, body.TopicTH, body.TopicEN, body.Code).Scan(&newTopicID)
		if err != nil {
			log.Printf("Error executing insert query: %v\nQuery: %s\nParams: %s, %s, %s", err, query, body.TopicTH, body.TopicEN, body.Code)
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to create topic")
			return
		}

		var newTopic models.Topic
		err = dbConn.QueryRow(`SELECT id, topic_th, topic_en, code FROM topics WHERE id = $1`, newTopicID).Scan(
			&newTopic.ID, &newTopic.TopicTH, &newTopic.TopicEN, &newTopic.Code)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve new topic")
			return
		}

		helpers.FormatSuccessResponse(c, newTopic)
	}
}

func UpdateTopic(dbConn *sql.DB, server *socketio.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var body struct {
			TopicTH *string `json:"topicTH"`
			TopicEN *string `json:"topicEN"`
			Code    *string `json:"code"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid request body")
			return
		}

		updateFields := []string{}
		updateValues := []interface{}{id}
		placeholderIndex := 2

		if body.TopicTH != nil {
			updateFields = append(updateFields, fmt.Sprintf("topic_th = $%d", placeholderIndex))
			updateValues = append(updateValues, *body.TopicTH)
			placeholderIndex++
		}
		if body.TopicEN != nil {
			updateFields = append(updateFields, fmt.Sprintf("topic_en = $%d", placeholderIndex))
			updateValues = append(updateValues, *body.TopicEN)
			placeholderIndex++
		}
		if body.Code != nil {
			updateFields = append(updateFields, fmt.Sprintf("code = $%d", placeholderIndex))
			updateValues = append(updateValues, *body.Code)
			placeholderIndex++
		}
		if len(updateFields) == 0 {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "No fields to update")
			return
		}

		query := "UPDATE topics SET " + helpers.Join(updateFields, ", ") + " WHERE id = $1"
		_, err := dbConn.Exec(query, updateValues...)
		if err != nil {
			if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
				helpers.FormatErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("The code '%v' already exists.", *body.Code))
			} else {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update topic")
			}
			return
		}

		var updatedTopic models.Topic
		err = dbConn.QueryRow(`SELECT id, topic_th, topic_en, code FROM topics WHERE id = $1`, id).Scan(
			&updatedTopic.ID, &updatedTopic.TopicTH, &updatedTopic.TopicEN, &updatedTopic.Code)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve updated topic")
			return
		}

		helpers.FormatSuccessResponse(c, updatedTopic)
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
