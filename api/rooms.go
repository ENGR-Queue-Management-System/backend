package api

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetRooms handles fetching all rooms
func GetRooms(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		rows, err := dbConn.Query("SELECT id, room FROM rooms")
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch rooms"})
		}
		defer rows.Close()

		var rooms []map[string]interface{}
		for rows.Next() {
			var id int
			var room string
			if err := rows.Scan(&id, &room); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read room data"})
			}
			rooms = append(rooms, map[string]interface{}{
				"id":   id,
				"room": room,
			})
		}

		if err := rows.Err(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error iterating rooms"})
		}

		return c.JSON(http.StatusOK, rooms)
	}
}

// CreateRoom handles creating a new room
func CreateRoom(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		room := new(struct {
			Room string `json:"room"`
		})

		if err := c.Bind(room); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid input"})
		}

		_, err := dbConn.Exec("INSERT INTO rooms (room) VALUES ($1) ON CONFLICT (room) DO NOTHING", room.Room)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create room"})
		}

		return c.JSON(http.StatusCreated, map[string]string{"message": "Room created successfully"})
	}
}
