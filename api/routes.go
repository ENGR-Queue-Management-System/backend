package api

import (
	"database/sql"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Group, db *sql.DB) {
	e.POST("/authentication", Authentication(db))

	e.GET("/user", GetUserInfo(db))

	e.GET("/counter", GetCounters(db))
	e.POST("/counter", CreateCounter(db))
	e.DELETE("/counter/:id", DeleteCounter(db))
}
