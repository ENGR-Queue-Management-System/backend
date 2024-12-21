package helpers

import (
	"fmt"
	"os"
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
	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey == "" {
		return "", fmt.Errorf("JWT secret key is not set")
	}
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("Invalid token")
	}

	claims, ok := token.Claims.(*jwt.MapClaims)
	if !ok {
		return "", fmt.Errorf("Invalid claims in token")
	}

	email, ok := (*claims)["email"].(string)
	if !ok {
		return "", fmt.Errorf("Email not found in token")
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
