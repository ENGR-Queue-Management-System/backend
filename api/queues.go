package api

import (
	"database/sql"
	"log"
	"net/http"
	"src/helpers"
	"src/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

func GetQueues(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("counter")
		if query == "" {
			query := `SELECT * FROM queues LEFT JOIN topics t ON topic_id = t.id`
			rows, err := dbConn.Query(query)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch queues"})
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
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read queue data"})
					return
				}
				queue.Topic = topic
				queues = append(queues, queue)
			}

			if err := rows.Err(); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iterating queues"})
				return
			}
			c.JSON(http.StatusOK, helpers.FormatSuccessResponse(queues))
		} else {
			counterID, err := strconv.Atoi(query)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Counter must be a valid integer"})
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
				ORDER BY created_at ASC;
			`
			waitingRows, err := dbConn.Query(waitingQueuesQuery, helpers.WAITING, counterID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch waiting queues"})
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
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read waiting queue data"})
					return
				}
				queue.Topic = topic
				waitingQueues = append(waitingQueues, queue)
			}

			currentQueueQuery := `SELECT * FROM queues WHERE status = $1 AND counter_id = $2;`
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current queue"})
				return
			} else {
				current = currentQueue
			}

			c.JSON(http.StatusOK, helpers.FormatSuccessResponse(map[string]interface{}{
				"queues":  waitingQueues,
				"current": current,
			}))
		}
	}
}

func GetStudentQueue(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		firstName := c.Query("firstName")
		lastName := c.Query("lastName")
		if firstName == "" || lastName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters: firstName and lastName"})
			return
		}

		var queue models.Queue
		var topic models.Topic
		queueQuery := `SELECT * FROM queues LEFT JOIN topics t ON topic_id = t.id WHERE firstname = $1 AND lastname = $2 AND status = $3`
		err := dbConn.QueryRow(queueQuery, firstName, lastName, helpers.WAITING).Scan(
			&queue.ID, &queue.No, &queue.StudentID, &queue.Firstname, &queue.Lastname,
			&queue.TopicID, &queue.Note, &queue.Status, &queue.CounterID, &queue.CreatedAt,
			&topic.ID, &topic.TopicTH, &topic.TopicEN, &topic.Code,
		)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusOK, helpers.FormatSuccessResponse(map[string]interface{}{"queue": map[string]interface{}{}}))
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve queue details"})
			return
		}
		queue.Topic = topic

		countWaitingAfterInProgress, err := FindWaitingQueue(dbConn, int(topic.ID), int(queue.ID), topic.Code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count waiting queues"})
			return
		}

		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(map[string]interface{}{
			"queue":   queue,
			"waiting": countWaitingAfterInProgress}))
	}
}

func CreateQueue(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, map[string]string{
			"message": "not create api",
		})
	}
}

func UpdateQueue(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusCreated, map[string]string{
			"message": "not create api",
		})
	}
}

func DeleteQueue(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		result, err := dbConn.Exec("DELETE FROM queues WHERE id = $1", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete queue"})
			return
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify deletion"})
			return
		}
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Queue not found"})
			return
		}
		c.JSON(http.StatusOK, map[string]string{"message": "Queue deleted successfully"})
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
