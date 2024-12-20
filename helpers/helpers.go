package helpers

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

func FormatSuccessResponse(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"message": "success",
		"data":    data,
	}
}

func ExtractEmailFromToken(c echo.Context) (string, error) {
	authHeader := c.Request().Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization header")
	}

	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["email"] == nil {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "Email not found in token")
	}

	email, ok := claims["email"].(string)
	if !ok {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "Invalid email in token")
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
