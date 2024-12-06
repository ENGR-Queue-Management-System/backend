package api

import (
	"database/sql"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Group, db *sql.DB) {
	// e.POST("/authentication", Authentication(db))
	e.GET("/rooms", GetRooms(db))
	e.POST("/rooms", CreateRoom(db))
}
