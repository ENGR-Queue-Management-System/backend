package helpers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func FormatSuccessResponse(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"message": "success",
		"data":    data,
	})
}

func FormatErrorResponse(c *gin.Context, statusCode int, data interface{}) {
	response := gin.H{
		"statusCode": statusCode,
		"status":     http.StatusText(statusCode),
	}
	if data != nil {
		switch v := data.(type) {
		case map[string]interface{}:
			for key, value := range v {
				response[key] = value
			}
		default:
			response["message"] = data
		}
	}
	c.JSON(statusCode, response)
}

func ExtractToken(c *gin.Context) (*jwt.MapClaims, error) {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("Invalid authorization header")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("Invalid token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("Invalid claims in token")
	}

	return &claims, nil
}

func GetBangkokTime() time.Time {
	loc := time.FixedZone("Asia/Bangkok", 7*60*60)
	return time.Now().In(loc)
}

func GetStartAndEndOfDay() (time.Time, time.Time) {
	t := GetBangkokTime()
	startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	return startOfDay, endOfDay
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
