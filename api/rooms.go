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

// DeleteRoom handles deleting a room by its ID
func DeleteRoom(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		result, err := dbConn.Exec("DELETE FROM rooms WHERE id = $1", id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete room"})
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to verify deletion"})
		}
		if rowsAffected == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Room not found"})
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "Room deleted successfully"})
	}
}
