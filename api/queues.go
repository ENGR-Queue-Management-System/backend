package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"src/helpers"
	"src/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReserveDTO struct {
	Topic     int     `json:"topic" validate:"required"`
	Note      *string `json:"note"`
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
}

func GetQueues(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		counterID := c.Query("counter")

		today := time.Now().Format("2006-01-02")

		if counterID == "" {
			var queues []models.Queue
			if err := db.Preload("Topic").Where("AND DATE(created_at) = ?", today).Find(&queues).Error; err != nil {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch queues")
				return
			}
			helpers.FormatSuccessResponse(c, queues)
			return
		}

		var waitingQueues []models.Queue
		if err := db.Preload("Topic").
			Where("status = ? AND topic_id IN (SELECT topic_id FROM counter_topics WHERE counter_id = ?)", helpers.WAITING, counterID).
			Order("created_at ASC, no ASC").
			Find(&waitingQueues).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch waiting queues")
			return
		}

		helpers.FormatSuccessResponse(c, waitingQueues)
	}
}

func GetStudentQueue(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		firstName := c.Query("firstName")
		lastName := c.Query("lastName")
		if firstName == "" || lastName == "" {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Missing required parameters: firstName and lastName")
			return
		}

		today := time.Now().Format("2006-01-02")

		var queue models.Queue
		var topic models.Topic
		err := db.Preload("Topic").Where("firstname = ? AND lastname = ? AND DATE(created_at) = ?", firstName, lastName, today).Order("created_at DESC, no DESC").First(&queue).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				helpers.FormatSuccessResponse(c, map[string]interface{}{"queue": map[string]interface{}{}})
				return
			}
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve queue details")
			return
		}

		countWaitingAfterInProgress, err := FindWaitingQueue(db, int(topic.ID), int(queue.ID), topic.Code)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to count waiting queues")
			return
		}

		helpers.FormatSuccessResponse(c, map[string]interface{}{
			"queue":   queue,
			"waiting": countWaitingAfterInProgress})
	}
}

func GetCalledQueues(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var calledQueues []models.Queue
		if err := db.Preload("Topic").
			Where("status = ?", helpers.CALLED).
			Order("created_at DESC, no DESC").
			Find(&calledQueues).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch waiting queues")
			return
		}

		helpers.FormatSuccessResponse(c, calledQueues)
	}
}

func CreateQueue(db *gorm.DB, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body ReserveDTO
		if err := c.Bind(&body); err != nil || body.Topic == 0 {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid topic")
			return
		}

		var topic models.Topic
		err := db.First(&topic, body.Topic).Error
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve topic")
			return
		}

		today := time.Now().Format("2006-01-02")

		var lastQueueNo string
		err = db.Model(&models.Queue{}).Where("topic_id = ? AND DATE(created_at) = ?", body.Topic, today).Order("no DESC").Limit(1).Pluck("no", &lastQueueNo).Error
		if err != nil && err != gorm.ErrRecordNotFound {
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

		var note *string
		if body.Note == nil {
			note = nil
		} else {
			note = body.Note
		}

		var firstName, lastName string
		var studentID *string
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
			studentIDClaim, ok := (*claims)["studentId"].(string)
			if !ok {
				helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid studentId in token")
				return
			}
			studentID = &studentIDClaim
			firstName = firstNameClaim
			lastName = lastNameClaim
		}

		queue := models.Queue{
			No:        newQueueNo,
			StudentID: studentID,
			Firstname: firstName,
			Lastname:  lastName,
			TopicID:   body.Topic,
			Note:      note,
		}

		if err := db.Create(&queue).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to create queue")
			return
		}

		countWaitingAfterInProgress, err := FindWaitingQueue(db, body.Topic, queue.ID, topic.Code)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to count waiting queues")
			return
		}

		err = db.Model(&queue).Preload("Topic").First(&queue).Error
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve queue details")
			return
		}
		message, _ := json.Marshal(map[string]interface{}{
			"event": "addQueue",
			"data": map[string]interface{}{
				"queue":   queue,
				"waiting": countWaitingAfterInProgress,
			},
		})
		hub.broadcast <- message

		if body.FirstName != nil && body.LastName != nil {
			tokenString, err := generateJWTToken(body, true)
			if err != nil {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to generate JWT token")
				return
			}

			helpers.FormatSuccessResponse(c, map[string]interface{}{
				"token":   tokenString,
				"queue":   queue,
				"waiting": countWaitingAfterInProgress,
			})
			return
		}

		helpers.FormatSuccessResponse(c, map[string]interface{}{
			"queue":   queue,
			"waiting": countWaitingAfterInProgress,
		})
	}
}

func UpdateQueue(db *gorm.DB, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		body := new(struct {
			Counter int `json:"counter"`
			Current int `json:"current"`
		})
		if err := c.ShouldBindJSON(body); err != nil {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid request body")
			return
		}
		tx := db.Begin()
		if err := tx.Model(&models.Queue{}).Where("id = ?", body.Current).Update("status", helpers.CALLED).Error; err != nil {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update current queue to CALLED")
			return
		}
		if err := tx.Model(&models.Queue{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status":     helpers.IN_PROGRESS,
			"counter_id": body.Counter,
		}).Error; err != nil {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update queue to IN_PROGRESS")
			return
		}
		var currentQueue models.Queue
		if err := tx.Preload("Topic").First(&currentQueue, id).Error; err != nil {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch current queue")
			return
		}
		if err := tx.Commit().Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to commit transaction")
			return
		}

		message, _ := json.Marshal(map[string]interface{}{
			"event": "updateQueue",
			"data": map[string]interface{}{
				"current": currentQueue,
				"called":  body.Current,
			},
		})
		hub.broadcast <- message

		helpers.FormatSuccessResponse(c, currentQueue)
	}
}

func DeleteQueue(db *gorm.DB, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := db.Delete(&models.Queue{}, id).Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to delete queue")
			return
		}

		message, _ := json.Marshal(map[string]interface{}{
			"event": "deleteQueue",
			"data":  id,
		})
		hub.broadcast <- message

		helpers.FormatSuccessResponse(c, map[string]string{"message": "Queue deleted successfully"})
	}
}

func FindWaitingQueue(db *gorm.DB, topicID int, queueID int, topicCode string) (int, error) {
	var count int64
	if err := db.Model(&models.Queue{}).
		Where("topic_id = ? AND status = ? AND id != ? AND no LIKE ?", topicID, helpers.WAITING, queueID, topicCode+"%").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}
