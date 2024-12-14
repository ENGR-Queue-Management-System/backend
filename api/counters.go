package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"src/helpers"
	"src/models"
	"time"

	"github.com/labstack/echo/v4"
)

func GetCounters(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		query := `SELECT c.id, c.counter, c.status, c.time_closed,
    u.id, u.firstName_TH, u.lastName_TH, u.firstName_EN, u.lastName_EN, u.email FROM counters c LEFT JOIN users u ON c.id = u.counter_id`
		rows, err := dbConn.Query(query)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch counters"})
		}
		defer rows.Close()

		var counters []models.CounterWithUser
		for rows.Next() {
			var counter models.CounterWithUser
			var timeClosed time.Time
			var user models.UserOnly
			if err := rows.Scan(
				&counter.ID, &counter.Counter, &counter.Status, &timeClosed,
				&user.ID, &user.FirstNameTH, &user.LastNameTH, &user.FirstNameEN, &user.LastNameEN, &user.Email,
			); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read counter data"})
			}
			counter.TimeClosed = timeClosed.Format("HH:mm:ss")
			counter.User = user
			counters = append(counters, counter)
		}

		if err := rows.Err(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error iterating rooms"})
		}
		return c.JSON(http.StatusOK, helpers.FormatSuccessResponse(counters))
	}
}

func CreateCounter(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		body := new(struct {
			Email   string `json:"email"`
			Counter string `json:"counter"`
		})
		if err := c.Bind(body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}

		tx, err := dbConn.Begin()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to start transaction"})
		}
		defer func() {
			if p := recover(); p != nil {
				tx.Rollback()
			}
		}()

		var counterID int
		err = tx.QueryRow(
			`INSERT INTO counters (counter) VALUES ($1) 
			ON CONFLICT (counter) DO UPDATE SET counter = EXCLUDED.counter
			RETURNING id`,
			body.Counter,
		).Scan(&counterID)
		if err != nil {
			tx.Rollback()
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create or fetch counter"})
		}
		_, err = tx.Exec(
			`INSERT INTO users (email, counter_id) VALUES ($1, $2)
				ON CONFLICT (email) DO UPDATE SET counter_id = EXCLUDED.counter_id`,
			body.Email, counterID,
		)
		if err != nil {
			tx.Rollback()
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
		}
		if err := tx.Commit(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to commit transaction"})
		}

		return c.JSON(http.StatusCreated, map[string]string{
			"message":   "Counter and user created successfully",
			"counterId": fmt.Sprintf("%d", counterID),
		})
	}
}

func DeleteCounter(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")
		result, err := dbConn.Exec("DELETE FROM counters WHERE id = $1", id)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete counter"})
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to verify deletion"})
		}
		if rowsAffected == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Counter not found"})
		}
		return c.JSON(http.StatusOK, map[string]string{"message": "Counter deleted successfully"})
	}
}
