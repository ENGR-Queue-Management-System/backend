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
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve notification schedule")
			return
		}
		helpers.FormatSuccessResponse(c, notiSchedule)
	}
}

func CreateNotiSchedule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body models.NotiSchedule
		if err := c.ShouldBindJSON(&body); err != nil {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid request body")
			return
		}

		if err := db.Model(&models.NotiSchedule{}).Create(&body).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to create notification schedule")
			return
		}

		helpers.FormatSuccessResponse(c, body)
	}
}

func UpdateNotiSchedule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var body models.NotiSchedule
		if err := c.ShouldBindJSON(&body); err != nil {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid request body")
			return
		}

		var notiSchedule models.NotiSchedule
		if err := db.First(&notiSchedule, id).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusNotFound, "Notification schedule not found")
			return
		}

		if err := db.Model(&notiSchedule).Updates(body).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update notification schedule")
			return
		}

		helpers.FormatSuccessResponse(c, notiSchedule)
	}
}

func DeleteNotiSchedule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := db.Delete(&models.NotiSchedule{}, id).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusNotFound, "Notification schedule not found")
			return
		}

		helpers.FormatSuccessResponse(c, map[string]string{"message": "Notification schedule deleted successfully"})
	}
}
