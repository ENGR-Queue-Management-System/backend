package api

import (
	"fmt"
	"log"
	"net/http"
	"src/helpers"
	"src/models"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetCounters(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var counters []models.Counter
		err := db.Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("ID", "CounterID", "FirstNameTH", "FirstNameEN", "LastNameTH", "LastNameEN", "Email")
		}).Preload("Topics", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).Order("counter ASC").Find(&counters).Error
		if err != nil {
			log.Println("Error fetching counters:", err)
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch counters")
			return
		}
		var response []models.CounterResponse
		for _, counter := range counters {
			var currentQueue *models.Queue
			err := db.Where("status = ? AND counter_id = ?", helpers.IN_PROGRESS, counter.ID).First(&currentQueue).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					currentQueue = nil
				} else {
					log.Println("Error fetching current queue for counter:", err)
					helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch current queue")
					return
				}
			}
			response = append(response, models.CounterResponse{
				ID:         counter.ID,
				Counter:    counter.Counter,
				Status:     counter.Status,
				TimeClosed: counter.TimeClosed,
				User: models.UserWithoutCounter{
					ID:          counter.User.ID,
					FirstNameTH: counter.User.FirstNameTH,
					LastNameTH:  counter.User.LastNameTH,
					FirstNameEN: counter.User.FirstNameEN,
					LastNameEN:  counter.User.LastNameEN,
					Email:       counter.User.Email,
				},
				Topics:       counter.Topics,
				CurrentQueue: currentQueue,
			})
		}
		helpers.FormatSuccessResponse(c, response)
	}
}

func CreateCounter(db *gorm.DB, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		body := new(struct {
			Counter    string `json:"counter"`
			Email      string `json:"email"`
			TimeClosed string `json:"timeClosed"`
			Topics     []int  `json:"topics"`
		})
		if err := c.Bind(body); err != nil {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid request body")
			return
		}

		tx := db.Begin()
		if tx.Error != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to start transaction")
			return
		}
		defer func() {
			if p := recover(); p != nil {
				tx.Rollback()
			}
		}()

		var counter models.Counter
		err := tx.Where("counter = ?", body.Counter).First(&counter).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Counter already exists")
			return
		}
		if err != gorm.ErrRecordNotFound {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch counter")
			return
		}
		counter = models.Counter{
			Counter:    body.Counter,
			TimeClosed: body.TimeClosed,
		}
		err = tx.Create(&counter).Error
		if err != nil {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to create counter")
			return
		}

		var user models.User
		err = tx.Where("email = ?", body.Email).First(&user).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to create user")
			return
		}
		if err == gorm.ErrRecordNotFound {
			user = models.User{
				Email:     body.Email,
				CounterID: counter.ID,
			}
			err = tx.Create(&user).Error
			if err != nil {
				tx.Rollback()
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to create user")
				return
			}
		} else {
			user.CounterID = counter.ID
			err = tx.Save(&user).Error
			if err != nil {
				tx.Rollback()
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update user")
				return
			}
		}

		for _, topicID := range body.Topics {
			var counterTopic models.CounterTopic
			err := tx.Where("counter_id = ? AND topic_id = ?", counter.ID, topicID).FirstOrCreate(&counterTopic, models.CounterTopic{CounterID: counter.ID, TopicID: topicID}).Error
			if err != nil {
				tx.Rollback()
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to associate topic %d with counter", topicID))
				return
			}
		}

		if err := tx.Commit().Error; err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to commit transaction")
			return
		}

		var result models.Counter
		err = db.Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("ID", "CounterID", "FirstNameTH", "FirstNameEN", "LastNameTH", "LastNameEN", "Email")
		}).Preload("Topics", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).Where("id = ?", counter.ID).First(&result).Error
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch counter data")
			return
		}

		// message, _ := json.Marshal(map[string]interface{}{
		// 	"event": "addCounter",
		// 	"data":  result,
		// })
		// hub.broadcast <- message

		helpers.FormatSuccessResponse(c, result)
	}
}

func UpdateCounter(db *gorm.DB, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid ID format")
			return
		}
		body := new(struct {
			Counter    *string `json:"counter"`
			Status     *bool   `json:"status"`
			TimeClosed *string `json:"timeClosed"`
			Email      *string `json:"email"`
			Topics     *[]int  `json:"topics"`
		})
		if err := c.ShouldBindJSON(&body); err != nil {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid request body")
			return
		}

		tx := db.Begin()
		if tx.Error != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to start transaction")
			return
		}
		defer func() {
			if p := recover(); p != nil {
				tx.Rollback()
			}
		}()

		var counter models.Counter
		err = tx.First(&counter, id).Error
		if err != nil {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Counter not found")
			return
		}

		if body.Counter != nil {
			counter.Counter = *body.Counter
		}
		if body.Status != nil {
			counter.Status = *body.Status
		}
		if body.TimeClosed != nil {
			counter.TimeClosed = *body.TimeClosed
		}

		err = tx.Save(&counter).Error
		if err != nil {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update counter")
			return
		}

		if body.Topics != nil {
			err := tx.Where("counter_id = ?", counter.ID).Delete(&models.CounterTopic{}).Error
			if err != nil {
				tx.Rollback()
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to remove old topics")
				return
			}
			for _, topicID := range *body.Topics {
				err := tx.Create(&models.CounterTopic{
					CounterID: counter.ID,
					TopicID:   topicID,
				}).Error
				if err != nil {
					tx.Rollback()
					helpers.FormatErrorResponse(c, http.StatusInternalServerError, fmt.Sprintf("Failed to associate topic %d with counter", topicID))
					return
				}
			}
		}

		if body.Email != nil {
			var user models.User
			err := tx.Where("email = ?", *body.Email).First(&user).Error
			if err != nil && err != gorm.ErrRecordNotFound {
				tx.Rollback()
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to check user")
				return
			}
			if err == gorm.ErrRecordNotFound {
				user = models.User{
					Email:     *body.Email,
					CounterID: counter.ID,
				}
				err = tx.Create(&user).Error
				if err != nil {
					tx.Rollback()
					helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to create user")
					return
				}
			} else {
				user.CounterID = counter.ID
				err = tx.Save(&user).Error
				if err != nil {
					tx.Rollback()
					helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update user with counter_id")
					return
				}
			}
		}

		err = tx.Commit().Error
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to commit transaction")
			return
		}

		var updatedCounter models.Counter
		err = db.Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("ID", "CounterID", "FirstNameTH", "FirstNameEN", "LastNameTH", "LastNameEN", "Email")
		}).Preload("Topics", func(db *gorm.DB) *gorm.DB { return db.Order("id ASC") }).Where("id = ?", counter.ID).First(&updatedCounter).Error
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to fetch updated counter data")
			return
		}

		// message, _ := json.Marshal(map[string]interface{}{
		// 	"event": "updateCounter",
		// 	"data":  updatedCounter,
		// })
		// hub.broadcast <- message

		helpers.FormatSuccessResponse(c, updatedCounter)
	}
}

func DeleteCounter(db *gorm.DB, hub *Hub) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		tx := db.Begin()
		if err := tx.Model(&models.User{}).Where("counter_id = ?", id).Update("counter_id", nil).Error; err != nil {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update associated users")
			return
		}
		if err := tx.Delete(&models.Counter{}, id).Error; err != nil {
			tx.Rollback()
			helpers.FormatErrorResponse(c, http.StatusNotFound, "Counter not found")
			return
		}
		tx.Commit()

		// message, _ := json.Marshal(map[string]interface{}{
		// 	"event": "deleteCounter",
		// 	"data":  id,
		// })
		// hub.broadcast <- message

		helpers.FormatSuccessResponse(c, map[string]string{"message": "Counter deleted successfully"})
	}
}
