package api

import (
	"net/http"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetUserInfo(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userClaims, ok := helpers.ExtractClaims(c)
		if !ok {
			return
		}
		email, ok := userClaims["email"].(string)
		if !ok || email == "" {
			helpers.FormatErrorResponse(c, http.StatusUnauthorized, "Email claim is missing or invalid in token")
			return
		}

		var user models.User
		err := db.Preload("Counter", func(db *gorm.DB) *gorm.DB {
			return db.Select("ID", "Counter", "TimeClosed", "Status")
		}).Where("email = ?", email).First(&user).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				helpers.FormatErrorResponse(c, http.StatusNotFound, "User not found")
			} else {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve user")
			}
			return
		}

		helpers.FormatSuccessResponse(c, user)
	}
}
