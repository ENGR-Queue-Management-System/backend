package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"src/helpers"
	"src/models"
	"strconv"

	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
)

type ReserveDTO struct {
	Topic     int     `json:"topic" validate:"required"`
	Note      *string `json:"note"`
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
}

func GetQueues(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("counter")
		if query == "" {
			query := `SELECT * FROM queues LEFT JOIN topics t ON topic_id = t.id`
			rows, err := dbConn.Query(query)
			if err != nil {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch queues")
				return
			}
			defer rows.Close()

			var queues []models.Queue = []models.Queue{}
			for rows.Next() {
				var queue models.Queue
				var topic models.Topic
				if err := rows.Scan(
					&queue.ID, &queue.No, &queue.StudentID, &queue.Firstname, &queue.Lastname,
					&queue.TopicID, &queue.Note, &queue.Status, &queue.CounterID, &queue.CreatedAt,
					&topic.ID, &topic.TopicTH, &topic.TopicEN, &topic.Code,
				); err != nil {
					helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to read queue data")
					return
				}
				queue.Topic = topic
				queues = append(queues, queue)
			}

			if err := rows.Err(); err != nil {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Error iterating queues")
				return
			}
			helpers.FormatSuccessResponse(c, queues)
		} else {
			counterID, err := strconv.Atoi(query)
			if err != nil {
				helpers.FormatErrorResponse(c, http.StatusBadRequest, "Counter must be a valid integer")
				return
			}
			waitingQueuesQuery := `
				SELECT * 
				FROM queues
				LEFT JOIN topics t ON topic_id = t.id
				WHERE status = $1
					AND topic_id IN (
							SELECT topic_id 
							FROM counter_topics 
							WHERE counter_id = $2
					)
				ORDER BY created_at ASC, no ASC;
			`
			waitingRows, err := dbConn.Query(waitingQueuesQuery, helpers.WAITING, counterID)
			if err != nil {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch waiting queues")
				return
			}
			defer waitingRows.Close()

			var waitingQueues []models.Queue = []models.Queue{}
			for waitingRows.Next() {
				var queue models.Queue
				var topic models.Topic
				if err := waitingRows.Scan(
					&queue.ID, &queue.No, &queue.StudentID, &queue.Firstname, &queue.Lastname,
					&queue.TopicID, &queue.Note, &queue.Status, &queue.CounterID, &queue.CreatedAt,
					&topic.ID, &topic.TopicTH, &topic.TopicEN, &topic.Code,
				); err != nil {
					helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to read waiting queue data")
					return
				}
				queue.Topic = topic
				waitingQueues = append(waitingQueues, queue)
			}

			currentQueueQuery := `SELECT * FROM queues WHERE status = $1 AND counter_id = $2 LIMIT 1;`
			var currentQueue models.Queue
			err = dbConn.QueryRow(currentQueueQuery, helpers.IN_PROGRESS, counterID).Scan(
				&currentQueue.ID, &currentQueue.No, &currentQueue.StudentID, &currentQueue.Firstname,
				&currentQueue.Lastname, &currentQueue.TopicID, &currentQueue.Note,
				&currentQueue.Status, &currentQueue.CounterID, &currentQueue.CreatedAt,
			)
			var current interface{}
			if err == sql.ErrNoRows {
				current = map[string]interface{}{}
			} else if err != nil {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch current queue")
				return
			} else {
				current = currentQueue
			}

			helpers.FormatSuccessResponse(c, map[string]interface{}{
				"queues":  waitingQueues,
				"current": current,
			})
		}
	}
}

func GetStudentQueue(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		firstName := c.Query("firstName")
		lastName := c.Query("lastName")
		if firstName == "" || lastName == "" {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Missing required parameters: firstName and lastName")
			return
		}

		var queue models.Queue = models.Queue{}
		var topic models.Topic
		queueQuery := `SELECT * FROM queues LEFT JOIN topics t ON topic_id = t.id WHERE firstname = $1 AND lastname = $2 AND status = $3`
		err := dbConn.QueryRow(queueQuery, firstName, lastName, helpers.WAITING).Scan(
			&queue.ID, &queue.No, &queue.StudentID, &queue.Firstname, &queue.Lastname,
			&queue.TopicID, &queue.Note, &queue.Status, &queue.CounterID, &queue.CreatedAt,
			&topic.ID, &topic.TopicTH, &topic.TopicEN, &topic.Code,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				helpers.FormatSuccessResponse(c, map[string]interface{}{"queue": map[string]interface{}{}})
				return
			}
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve queue details")
			return
		}
		queue.Topic = topic

		countWaitingAfterInProgress, err := FindWaitingQueue(dbConn, int(topic.ID), int(queue.ID), topic.Code)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to count waiting queues")
			return
		}

		helpers.FormatSuccessResponse(c, map[string]interface{}{
			"queue":   queue,
			"waiting": countWaitingAfterInProgress})
	}
}

func CreateQueue(dbConn *sql.DB, server *socketio.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body ReserveDTO
		if err := c.Bind(&body); err != nil || body.Topic == 0 {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid topic")
			return
		}

		var topic models.Topic
		topicQuery := `SELECT * FROM topics WHERE id = $1`
		err := dbConn.QueryRow(topicQuery, body.Topic).Scan(&topic.ID, &topic.TopicTH, &topic.TopicEN, &topic.Code)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve topic")
			return
		}
		var lastQueueNo string
		query := `SELECT no FROM queues WHERE topic_id = $1 ORDER BY no DESC LIMIT 1`
		err = dbConn.QueryRow(query, body.Topic).Scan(&lastQueueNo)
		if err != nil && err != sql.ErrNoRows {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve the last queue number")
			return
		}
		var newQueueNo string
		if lastQueueNo != "" {
			var numPart int
			_, err := fmt.Sscanf(lastQueueNo, topic.Code+"%03d", &numPart)
			if err != nil {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to parse the last queue number")
				return
			}
			numPart++
			newQueueNo = fmt.Sprintf("%s%03d", topic.Code, numPart)
		} else {
			newQueueNo = fmt.Sprintf("%s001", topic.Code)
		}

		var note interface{}
		if body.Note == nil {
			note = nil
		} else {
			note = *body.Note
		}

		var firstName string
		var lastName string
		var studentId *string
		if body.FirstName != nil && body.LastName != nil {
			firstName = *body.FirstName
			lastName = *body.LastName
		} else {
			claims, err := helpers.ExtractToken(c)
			if err != nil {
				helpers.FormatErrorResponse(c, http.StatusUnauthorized, err.Error())
				return
			}
			firstNameClaim, ok := (*claims)["firstName"].(string)
			if !ok {
				helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid firstName in token")
				return
			}
			lastNameClaim, ok := (*claims)["lastName"].(string)
			if !ok {
				helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid lastName in token")
				return
			}
			studentIdClaim := (*claims)["studentId"].(string)
			if !ok {
				helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid studentId in token")
				return
			}
			studentId = &studentIdClaim
			firstName = firstNameClaim
			lastName = lastNameClaim
		}

		insertQuery := `INSERT INTO queues (no, student_id, firstName, lastName, topic_id, note)
						VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
		var queueID int
		err = dbConn.QueryRow(insertQuery, newQueueNo, studentId, firstName, lastName, body.Topic, note).Scan(&queueID)
		if err != nil {
			log.Printf("Error inserting queue: %v", err)
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to create queue")
			return
		}

		countWaitingAfterInProgress, err := FindWaitingQueue(dbConn, body.Topic, queueID, topic.Code)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to count waiting queues")
			return
		}

		var queue models.Queue
		queueQuery := `SELECT * FROM queues q WHERE id = $1`
		err = dbConn.QueryRow(queueQuery, queueID).Scan(&queue.ID, &queue.No, &queue.StudentID, &queue.Firstname, &queue.Lastname, &queue.TopicID, &queue.Note, &queue.Status, &queue.CounterID, &queue.CreatedAt)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve queue details")
			return
		}
		queue.Topic = topic

		if body.FirstName != nil && body.LastName != nil {
			tokenString, err := generateJWTToken(body, true)
			if err != nil {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to generate JWT token")
				return
			}

			// server.BroadcastToNamespace(helpers.SOCKET, "addQueue", queue)
			helpers.FormatSuccessResponse(c, map[string]interface{}{
				"token":   tokenString,
				"queue":   queue,
				"waiting": countWaitingAfterInProgress,
			})
			return
		}

		// server.BroadcastToNamespace(helpers.SOCKET, "addQueue", queue)
		helpers.FormatSuccessResponse(c, map[string]interface{}{
			"queue":   queue,
			"waiting": countWaitingAfterInProgress,
		})
	}
}

func UpdateQueue(dbConn *sql.DB, server *socketio.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		body := new(struct {
			Counter int `json:"counter"`
		})
		if err := c.ShouldBindJSON(body); err != nil {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid request body")
			return
		}
		tx, err := dbConn.Begin()
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to start transaction")
			return
		}
		updateCalledQuery := `UPDATE queues SET status = $1 WHERE status = $2 AND counter_id = $3;`
		_, err = tx.Exec(updateCalledQuery, helpers.CALLED, helpers.IN_PROGRESS, body.Counter)
		if err != nil {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update current queue to CALLED")
			return
		}
		updateInProgressQuery := `UPDATE queues SET status = $1, counter_id = $2 WHERE id = $3;`
		_, err = tx.Exec(updateInProgressQuery, helpers.IN_PROGRESS, body.Counter, id)
		if err != nil {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update queue to IN_PROGRESS")
			return
		}
		currentQueueQuery := `SELECT * FROM queues WHERE id = $1;`
		var currentQueue models.Queue
		err = tx.QueryRow(currentQueueQuery, id).Scan(
			&currentQueue.ID, &currentQueue.No, &currentQueue.StudentID, &currentQueue.Firstname,
			&currentQueue.Lastname, &currentQueue.TopicID, &currentQueue.Note,
			&currentQueue.Status, &currentQueue.CounterID, &currentQueue.CreatedAt,
		)
		if err != nil {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch current queue")
			return
		}
		if err := tx.Commit(); err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to commit transaction")
			return
		}
		helpers.FormatSuccessResponse(c, currentQueue)
	}
}

func DeleteQueue(dbConn *sql.DB, server *socketio.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := dbConn.Exec("DELETE FROM queues WHERE id = $1", id)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to delete queue")
			return
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to verify deletion")
			return
		}
		if rowsAffected == 0 {
			helpers.FormatErrorResponse(c, http.StatusNotFound, "Queue not found")
			return
		}
		helpers.FormatSuccessResponse(c, map[string]string{"message": "Queue deleted successfully"})
	}
}

func FindWaitingQueue(dbConn *sql.DB, topicID int, queueID int, topicCode string) (int, error) {
	var lastInProgressQueueNo string
	inProgressQuery := `
			SELECT no 
			FROM queues 
			WHERE topic_id = $1 
			AND status = 'IN_PROGRESS' 
			ORDER BY created_at DESC 
			LIMIT 1
		`
	err := dbConn.QueryRow(inProgressQuery, topicID).Scan(&lastInProgressQueueNo)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error retrieving the last 'IN_PROGRESS' queue: %v", err)
		return -1, err
	}

	var countWaitingAfterInProgress int
	var countWaitingQuery string
	if lastInProgressQueueNo != "" {
		countWaitingQuery = `
				SELECT COUNT(*) 
				FROM queues 
				WHERE topic_id = $1 
				AND status = 'WAITING' 
				AND no > $2
				AND id != $3
				AND no LIKE $4
			`
		err = dbConn.QueryRow(countWaitingQuery, topicID, lastInProgressQueueNo, queueID, topicCode+"%").Scan(&countWaitingAfterInProgress)
	} else {
		countWaitingQuery = `
				SELECT COUNT(*) 
				FROM queues 
				WHERE topic_id = $1 
				AND status = 'WAITING'
				AND id != $2
				AND no LIKE $3
			`
		err = dbConn.QueryRow(countWaitingQuery, topicID, queueID, topicCode+"%").Scan(&countWaitingAfterInProgress)
	}
	if err != nil {
		log.Printf("Error counting waiting queues after the 'IN_PROGRESS' queue: %v", err)
		return -1, err
	}
	return countWaitingAfterInProgress, nil
}
