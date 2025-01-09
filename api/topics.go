package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetTopics(dbConn *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var topics []struct {
			ID      int    `json:"id"`
			TopicTH string `json:"topicTH"`
			TopicEN string `json:"topicEN"`
			Code    string `json:"code"`
			Waiting int    `json:"waiting"`
		}
		if err := dbConn.Table("topics").
			Select("topics.id, topics.topic_th, topics.topic_en, topics.code, COUNT(queues.id) AS waiting").
			Joins("LEFT JOIN queues ON queues.topic_id = topics.id AND queues.status IN (?, ?)", helpers.WAITING, helpers.IN_PROGRESS).
			Group("topics.id").
			Order("topics.id ASC").
			Scan(&topics).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch topics with waiting queues")
			return
		}
		helpers.FormatSuccessResponse(c, topics)
	}
}

func CreateTopic(dbConn *gorm.DB, hub *Hub) gin.HandlerFunc {
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

		var existingTopic models.Topic
		if err := dbConn.Where("code = ?", body.Code).First(&existingTopic).Error; err == nil {
			helpers.FormatErrorResponse(c, http.StatusConflict, fmt.Sprintf("The code '%v' already exists.", body.Code))
			return
		} else if err != gorm.ErrRecordNotFound {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to check for existing topic")
			return
		}

		topic := models.Topic{
			TopicTH: body.TopicTH,
			TopicEN: body.TopicEN,
			Code:    body.Code,
		}
		if err := dbConn.Create(&topic).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to create topic")
			return
		}

		// message, _ := json.Marshal(map[string]interface{}{
		// 	"event": "addTopic",
		// 	"data":  topic,
		// })
		// hub.broadcast <- message

		helpers.FormatSuccessResponse(c, topic)
	}
}

func UpdateTopic(dbConn *gorm.DB, hub *Hub) gin.HandlerFunc {
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

		var topic models.Topic
		if err := dbConn.First(&topic, id).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusNotFound, "Topic not found")
			return
		}

		if body.Code != nil {
			var existingTopic models.Topic
			if err := dbConn.Where("code = ?", *body.Code).First(&existingTopic).Error; err == nil {
				helpers.FormatErrorResponse(c, http.StatusConflict, fmt.Sprintf("The code '%v' already exists.", *body.Code))
				return
			}
			topic.Code = *body.Code
		}
		if body.TopicTH != nil {
			topic.TopicTH = *body.TopicTH
		}
		if body.TopicEN != nil {
			topic.TopicEN = *body.TopicEN
		}

		if err := dbConn.Save(&topic).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update topic")
			return
		}

		message, _ := json.Marshal(map[string]interface{}{
			"event": "updateTopic",
			"data":  topic,
		})
		hub.broadcast <- message

		helpers.FormatSuccessResponse(c, topic)
	}
}

func DeleteTopic(dbConn *gorm.DB, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := dbConn.Delete(&models.Topic{}, id).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusNotFound, "Topic not found")
			return
		}

		// message, _ := json.Marshal(map[string]interface{}{
		// 	"event": "deleteTopic",
		// 	"data":  id,
		// })
		// hub.broadcast <- message

		helpers.FormatSuccessResponse(c, map[string]string{"message": "Topic deleted successfully"})
	}
}
