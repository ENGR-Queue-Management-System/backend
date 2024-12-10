package api

import (
	"database/sql"

	"github.com/labstack/echo/v4"
)

func RegisterRoutes(e *echo.Group, db *sql.DB) {
	e.POST("/authentication", Authentication(db))

	e.GET("/user", GetUserInfo(db))

	e.GET("/room", GetRooms(db))
	e.POST("/room", CreateRoom(db))
	e.DELETE("/room/:id", DeleteRoom(db))
}
