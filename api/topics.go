package api

import (
	"fmt"
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"gorm.io/gorm"
)

func GetTopics(dbConn *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var topics []models.Topic
		if err := dbConn.Find(&topics).Order("id ASC").Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch topics")
			return
		}
		helpers.FormatSuccessResponse(c, topics)
	}
}

func CreateTopic(dbConn *gorm.DB, server *socketio.Server) gin.HandlerFunc {
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

		helpers.FormatSuccessResponse(c, topic)
	}
}

func UpdateTopic(dbConn *gorm.DB, server *socketio.Server) gin.HandlerFunc {
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

		helpers.FormatSuccessResponse(c, topic)
	}
}

func DeleteTopic(dbConn *gorm.DB, server *socketio.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := dbConn.Delete(&models.Topic{}, id).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusNotFound, "Topic not found")
			return
		}

		helpers.FormatSuccessResponse(c, map[string]string{"message": "Topic deleted successfully"})
	}
}
