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

	"github.com/gin-gonic/gin"
)

func GetCounters(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch counters"})
			return
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read counter data"})
				return
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating counters"})
			return
		}

		var counters []models.CounterWithUserWithTopics
		for _, counter := range countersMap {
			counters = append(counters, *counter)
		}
		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(counters))
	}
}

func CreateCounter(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := new(struct {
			Email   string `json:"email"`
			Counter string `json:"counter"`
		})
		if err := c.Bind(body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		tx, err := dbConn.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create or fetch counter"})
			return
		}
		_, err = tx.Exec(
			`INSERT INTO users (email, counter_id) VALUES ($1, $2)
				ON CONFLICT (email) DO UPDATE SET counter_id = EXCLUDED.counter_id`,
			body.Email, counterID,
		)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusCreated, helpers.FormatSuccessResponse(map[string]string{
			"message":   "Counter and user created successfully",
			"counterId": fmt.Sprintf("%d", counterID),
		}))
	}
}

func UpdateCounter(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
			return
		}
		body := new(struct {
			Counter    *string `json:"counter"`
			Status     *bool   `json:"status"`
			TimeClosed *string `json:"timeClosed"`
			Email      *string `json:"email"`
		})
		if err := c.Bind(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
		updateFields := []string{}
		updateValues := []interface{}{}
		placeholderIndex := 2
		if body.Counter != nil {
			updateFields = append(updateFields, "counter = $"+strconv.Itoa(placeholderIndex))
			updateValues = append(updateValues, *body.Counter)
			placeholderIndex++
		}
		if body.Status != nil {
			var statusValue int
			if *body.Status {
				statusValue = 1
			} else {
				statusValue = 0
			}
			updateFields = append(updateFields, "status = $"+strconv.Itoa(placeholderIndex))
			updateValues = append(updateValues, statusValue)
			placeholderIndex++
		}
		if body.TimeClosed != nil {
			updateFields = append(updateFields, "time_closed = $"+strconv.Itoa(placeholderIndex))
			updateValues = append(updateValues, *body.TimeClosed)
			placeholderIndex++
		}
		if len(updateFields) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
			return
		}

		query := "UPDATE counters SET " + helpers.Join(updateFields, ", ") + " WHERE id = $1"
		updateValues = append(updateValues, id)
		_, err = dbConn.Exec(query, updateValues...)
		if err != nil {
			log.Printf("Error executing query: %v, Query: %s, Values: %v", err, query, updateValues)

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update counter"})
			return
		}
		if body.Email != nil {
			var userID int64
			selectQuery := "SELECT id FROM users WHERE email = $1"
			err := dbConn.QueryRow(selectQuery, *body.Email).Scan(&userID)
			if err != nil && err != sql.ErrNoRows {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user"})
				return
			}
			if err == sql.ErrNoRows {
				insertQuery := "INSERT INTO users (email, counter_id) VALUES ($1, $2)"
				result, err := dbConn.Exec(insertQuery, *body.Email, id)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
					return
				}
				userID, err = result.LastInsertId()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get new user ID"})
					return
				}
			} else {
				updateUserQuery := "UPDATE users SET counter_id = $1 WHERE id = $2"
				_, err = dbConn.Exec(updateUserQuery, id, userID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user with counter_id"})
					return
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch updated counter"})
			return
		}
		updatedCounter.TimeClosed = timeClosed.Format("15:04:05")
		updatedCounter.User = user
		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(updatedCounter))
	}
}

func DeleteCounter(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := dbConn.Exec("DELETE FROM counters WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete counter"})
			return
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify deletion"})
			return
		}
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Counter not found"})
			return
		}
		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(map[string]string{"message": "Counter deleted successfully"}))
	}
}
