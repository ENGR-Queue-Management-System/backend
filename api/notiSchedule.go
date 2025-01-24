package api

import (
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetNotiSchedule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var notiSchedule []models.NotiSchedule
		if err := db.Find(&notiSchedule).Error; err != nil && err != gorm.ErrRecordNotFound {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve notiSchedule")
			return
		}
		helpers.FormatSuccessResponse(c, notiSchedule)
	}
}

func CreateNotiSchedule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body models.NotiSchedule
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		if err := db.Model(&models.NotiSchedule{}).Create(&body).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update config")
			return
		}

		helpers.FormatSuccessResponse(c, body)
	}
}
