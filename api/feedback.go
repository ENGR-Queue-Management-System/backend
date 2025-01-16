package api

import (
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

func GetFeedbackByUser(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := helpers.ExtractToken(c)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusUnauthorized, err.Error())
			return
		}
		email, ok := (*claims)["email"].(string)
		if !ok || email == "" {
			helpers.FormatErrorResponse(c, http.StatusUnauthorized, "Email claim is missing or invalid in token")
			return
		}
		var user models.User
		err = db.Where("email = ?", email).First(&user).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				helpers.FormatErrorResponse(c, http.StatusNotFound, "User not found")
			} else {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve user")
			}
			return
		}
		var feedback []models.Feedback
		err = db.Where("user_id = ?", user.ID).Find(&feedback).Error
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve feedback")
			return
		}
		helpers.FormatSuccessResponse(c, feedback)
	}
}

func CreateFeedback(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := new(struct {
			UserId   int      `json:"userId"`
			TopicId  int      `json:"topicId"`
			Rating   int      `json:"rating"`
			Tags     []string `json:"tags"`
			Feedback *string  `json:"feedback"`
		})
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		feedback := models.Feedback{
			UserID:   body.UserId,
			TopicID:  body.TopicId,
			Rating:   body.Rating,
			Tags:     pq.StringArray(body.Tags),
			Feedback: body.Feedback,
		}

		if err := db.Create(&feedback).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to create feedback"})
			return
		}

		helpers.FormatSuccessResponse(c, map[string]interface{}{"message": "Feedback created successfully"})
	}
}
