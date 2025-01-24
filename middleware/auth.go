package middleware

import (
	"net/http"
	"src/helpers"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			helpers.FormatErrorResponse(c, http.StatusUnauthorized, "Invalid authorization header")
			return
		}

		if !strings.HasPrefix(token, "Bearer ") {
			helpers.FormatErrorResponse(c, http.StatusUnauthorized, "Authorization token format is invalid")
			return
		}

		token = strings.TrimPrefix(token, "Bearer ")

		claims, err := helpers.VerifyToken(token)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusUnauthorized, err.Error())
			return
		}

		c.Set("claims", claims)
		c.Next()
	}
}
