package helpers

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func FormatSuccessResponse(data interface{}) gin.H {
	return gin.H{
		"message": "success",
		"data":    data,
	}
}

func ExtractEmailFromToken(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("Invalid authorization header")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("Invalid token")
	}
	claims := token.Claims.(jwt.MapClaims)
	email, ok := claims["email"].(string)
	if !ok {
		return "", fmt.Errorf("Invalid email in token")
	}

	return email, nil
}

func Capitalize(s string) string {
	if len(s) > 0 {
		return strings.ToUpper(string(s[0])) + s[1:]
	}
	return s
}

func Join(arr []string, separator string) string {
	result := ""
	for i, s := range arr {
		if i > 0 {
			result += separator
		}
		result += s
	}
	return result
}
