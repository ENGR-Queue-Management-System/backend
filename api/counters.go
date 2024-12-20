package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"src/helpers"
	"src/models"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

func GetCounters(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		query := `
		SELECT 
				c.id, c.counter, c.status, c.time_closed,
				u.id AS user_id, u.firstName_TH, u.lastName_TH, u.firstName_EN, u.lastName_EN, u.email,
				t.id AS topic_id, t.topic_th, t.topic_en, t.code
		FROM 
				counters c
		LEFT JOIN 
				users u ON c.id = u.counter_id
		LEFT JOIN 
				counter_topics ct ON c.id = ct.counter_id
		LEFT JOIN 
				topics t ON ct.topic_id = t.id
		ORDER BY 
				c.id, t.id;
		`
		rows, err := dbConn.Query(query)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch counters"})
		}
		defer rows.Close()

		countersMap := make(map[int]*models.CounterWithUserWithTopics)
		for rows.Next() {
			var counter models.CounterWithUserWithTopics
			var timeClosed time.Time
			var user models.UserOnly
			var topic models.Topic
			if err := rows.Scan(
				&counter.ID, &counter.Counter, &counter.Status, &timeClosed,
				&user.ID, &user.FirstNameTH, &user.LastNameTH, &user.FirstNameEN, &user.LastNameEN, &user.Email,
				&topic.ID, &topic.TopicTH, &topic.TopicEN, &topic.Code,
			); err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read counter data"})
			}
			counter.TimeClosed = timeClosed.Format("15:04:05")
			counter.User = user
			if existingCounter, exists := countersMap[counter.ID]; exists {
				existingCounter.Topic = append(existingCounter.Topic, topic)
			} else {
				counter.Topic = []models.Topic{topic}
				countersMap[counter.ID] = &counter
			}
		}

		if err := rows.Err(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error iterating counters"})
		}

		var counters []models.CounterWithUserWithTopics
		for _, counter := range countersMap {
			counters = append(counters, *counter)
		}
		return c.JSON(http.StatusOK, helpers.FormatSuccessResponse(counters))
	}
}

func CreateCounter(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		requestBody := new(struct {
			Email   string `json:"email"`
			Counter string `json:"counter"`
		})
		if err := c.Bind(requestBody); err != nil {
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
			requestBody.Counter,
		).Scan(&counterID)
		if err != nil {
			tx.Rollback()
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create or fetch counter"})
		}
		_, err = tx.Exec(
			`INSERT INTO users (email, counter_id) VALUES ($1, $2)
				ON CONFLICT (email) DO UPDATE SET counter_id = EXCLUDED.counter_id`,
			requestBody.Email, counterID,
		)
		if err != nil {
			tx.Rollback()
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
		}
		if err := tx.Commit(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to commit transaction"})
		}

		return c.JSON(http.StatusCreated, helpers.FormatSuccessResponse(map[string]string{
			"message":   "Counter and user created successfully",
			"counterId": fmt.Sprintf("%d", counterID),
		}))
	}
}

func UpdateCounter(dbConn *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
		}
		requestBody := new(struct {
			Counter    *string `json:"counter"`
			Status     *bool   `json:"status"`
			TimeClosed *string `json:"timeClosed"`
			Email      *string `json:"email"`
		})
		if err := c.Bind(&requestBody); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}
		updateFields := []string{}
		updateValues := []interface{}{}
		placeholderIndex := 2
		if requestBody.Counter != nil {
			updateFields = append(updateFields, "counter = $"+strconv.Itoa(placeholderIndex))
			updateValues = append(updateValues, *requestBody.Counter)
			placeholderIndex++
		}
		if requestBody.Status != nil {
			var statusValue int
			if *requestBody.Status {
				statusValue = 1
			} else {
				statusValue = 0
			}
			updateFields = append(updateFields, "status = $"+strconv.Itoa(placeholderIndex))
			updateValues = append(updateValues, statusValue)
			placeholderIndex++
		}
		if requestBody.TimeClosed != nil {
			updateFields = append(updateFields, "time_closed = $"+strconv.Itoa(placeholderIndex))
			updateValues = append(updateValues, *requestBody.TimeClosed)
			placeholderIndex++
		}
		if len(updateFields) == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "No fields to update"})
		}

		query := "UPDATE counters SET " + helpers.Join(updateFields, ", ") + " WHERE id = $1"
		updateValues = append(updateValues, id)
		_, err = dbConn.Exec(query, updateValues...)
		if err != nil {
			log.Printf("Error executing query: %v, Query: %s, Values: %v", err, query, updateValues)

			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update counter"})
		}
		if requestBody.Email != nil {
			var userID int64
			selectQuery := "SELECT id FROM users WHERE email = $1"
			err := dbConn.QueryRow(selectQuery, *requestBody.Email).Scan(&userID)
			if err != nil && err != sql.ErrNoRows {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check user"})
			}
			if err == sql.ErrNoRows {
				insertQuery := "INSERT INTO users (email, counter_id) VALUES ($1, $2)"
				result, err := dbConn.Exec(insertQuery, *requestBody.Email, id)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
				}
				userID, err = result.LastInsertId()
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get new user ID"})
				}
			} else {
				updateUserQuery := "UPDATE users SET counter_id = $1 WHERE id = $2"
				_, err = dbConn.Exec(updateUserQuery, id, userID)
				if err != nil {
					return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user with counter_id"})
				}
			}
		}
		var updatedCounter models.CounterWithUserWithTopics
		selectQuery := `SELECT c.id, c.counter, c.status, c.time_closed, 
                        u.id, u.firstName_TH, u.lastName_TH, u.firstName_EN, u.lastName_EN, u.email 
                        FROM counters c LEFT JOIN users u ON c.id = u.counter_id 
                        WHERE c.id = $1`
		row := dbConn.QueryRow(selectQuery, id)
		var timeClosed time.Time
		var user models.UserOnly
		if err := row.Scan(
			&updatedCounter.ID, &updatedCounter.Counter, &updatedCounter.Status, &timeClosed,
			&user.ID, &user.FirstNameTH, &user.LastNameTH, &user.FirstNameEN, &user.LastNameEN, &user.Email,
		); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch updated counter"})
		}
		updatedCounter.TimeClosed = timeClosed.Format("15:04:05")
		updatedCounter.User = user
		return c.JSON(http.StatusOK, helpers.FormatSuccessResponse(updatedCounter))
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
		return c.JSON(http.StatusOK, helpers.FormatSuccessResponse(map[string]string{"message": "Counter deleted successfully"}))
	}
}
