package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

type CMU_ENTRAID_ROLE int

const (
	MIS CMU_ENTRAID_ROLE = iota
	STUDENT
	ALUMNI
	RESIGN
	MANAGER
	NON_MIS
	ORG
	PROJECT
	RETIRED
	VIP
)

func (r CMU_ENTRAID_ROLE) String() string {
	return [...]string{
		"MISEmpAcc",
		"StdAcc",
		"AlumAcc",
		"EmpResiAcc",
		"ManAcc",
		"NonMISEmpAcc",
		"OrgAcc",
		"ProjAcc",
		"RetEmpAcc",
		"VIPAcc",
	}[r]
}

type LoginDTO struct {
	Topic     int     `json:"topic" validate:"required"`
	Note      *string `json:"note"`
	FirstName string  `json:"firstName" validate:"required"`
	LastName  string  `json:"lastName" validate:"required"`
}

type AuthDTO struct {
	Code        string `json:"code" validate:"required"`
	RedirectURI string `json:"redirectUri" validate:"required"`
}

type CmuEntraIDBasicInfoDTO struct {
	CmuitAccountName   string `json:"cmuitaccount_name"`
	CmuitAccount       string `json:"cmuitaccount"`
	StudentID          string `json:"student_id"`
	PrenameID          string `json:"prename_id"`
	PrenameTH          string `json:"prename_TH"`
	PrenameEN          string `json:"prename_EN"`
	FirstnameTH        string `json:"firstname_TH"`
	FirstnameEN        string `json:"firstname_EN"`
	LastnameTH         string `json:"lastname_TH"`
	LastnameEN         string `json:"lastname_EN"`
	OrganizationCode   string `json:"organization_code"`
	OrganizationNameTH string `json:"organization_name_TH"`
	OrganizationNameEN string `json:"organization_name_EN"`
	ItAccountTypeID    string `json:"itaccounttype_id"`
	ItAccountTypeTH    string `json:"itaccounttype_TH"`
	ItAccountTypeEN    string `json:"itaccounttype_EN"`
}

func getEntraIDAccessToken(code, redirectUri string) (string, error) {
	data := url.Values{}
	data.Set("code", code)
	data.Set("redirect_uri", redirectUri)
	data.Set("client_id", os.Getenv("CMU_ENTRAID_CLIENT_ID"))
	data.Set("client_secret", os.Getenv("CMU_ENTRAID_CLIENT_SECRET"))
	data.Set("scope", os.Getenv("SCOPE"))
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequest("POST", os.Getenv("CMU_ENTRAID_GET_TOKEN_URL"), bytes.NewBufferString(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes := new(bytes.Buffer)
		bodyBytes.ReadFrom(resp.Body)
		return "", fmt.Errorf("failed to fetch access token, status: %d, body: %s", resp.StatusCode, bodyBytes.String())
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	token, ok := result["access_token"].(string)
	if !ok {
		return "", errors.New("invalid access token response")
	}

	return token, nil
}

func getCMUBasicInfo(accessToken string) (*CmuEntraIDBasicInfoDTO, error) {
	client := &http.Client{}
	url := os.Getenv("CMU_ENTRAID_GET_BASIC_INFO")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to fetch CMU basic info")
	}

	var info CmuEntraIDBasicInfoDTO
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	return &info, nil
}

func generateJWTToken(user interface{}, notAdmin bool) (string, error) {
	var firstName, lastName string
	claims := jwt.MapClaims{}
	switch v := user.(type) {
	case CmuEntraIDBasicInfoDTO:
		claims["email"] = v.CmuitAccount
		firstName = v.FirstnameTH
		lastName = v.LastnameTH
		if v.StudentID != "" && notAdmin {
			claims["studentId"] = v.StudentID
		}
		if firstName == "" {
			firstName = helpers.Capitalize(v.FirstnameEN)
		}
		if lastName == "" {
			lastName = helpers.Capitalize(v.LastnameEN)
		}
		claims["faculty"] = v.OrganizationNameTH
	case LoginDTO:
		firstName = v.FirstName
		lastName = v.LastName
	}
	claims["firstName"] = firstName
	claims["lastName"] = lastName

	// expirationTime := time.Now().Add(7 * 24 * time.Hour)
	// claims["exp"] = expirationTime.Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	secretKey := os.Getenv("JWT_SECRET_KEY")
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func Authentication(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body AuthDTO
		if err := c.Bind(&body); err != nil || body.Code == "" || body.RedirectURI == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid authorization code or redirect URI"})
			return
		}
		accessToken, err := getEntraIDAccessToken(body.Code, body.RedirectURI)
		if err != nil || accessToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot get EntraID access token"})
			return
		}
		basicInfo, err := getCMUBasicInfo(accessToken)
		if err != nil || basicInfo == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot get CMU basic info"})
			return
		}

		row := dbConn.QueryRow("SELECT * FROM users WHERE email = $1", basicInfo.CmuitAccount)
		var user models.User
		err = row.Scan(&user.ID, &user.FirstNameTH, &user.LastNameTH, &user.FirstNameEN, &user.LastNameEN, &user.Email, &user.CounterID)
		if err == sql.ErrNoRows {
			if basicInfo.ItAccountTypeID == STUDENT.String() {
				tokenString, err := generateJWTToken(*basicInfo, true)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT token"})
					return
				}
				c.JSON(http.StatusOK, helpers.FormatSuccessResponse(map[string]interface{}{
					"token": tokenString,
				}))
				return
			} else {
				c.JSON(http.StatusForbidden, map[string]interface{}{
					"message": "Cannot access",
				})
				return
			}
		}
		tokenString, err := generateJWTToken(*basicInfo, false)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT token"})
			return
		}

		if user.FirstNameEN == nil || user.LastNameEN == nil {
			updateQuery := `UPDATE users SET firstname_th = $1, lastname_th = $2, firstname_en = $3, lastname_en = $4 WHERE email = $5`
			_, err := dbConn.Exec(updateQuery, basicInfo.FirstnameTH, basicInfo.LastnameTH, basicInfo.FirstnameEN, basicInfo.LastnameEN, basicInfo.CmuitAccount)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user data"})
				return
			}
			user.FirstNameTH = &basicInfo.FirstnameTH
			user.LastNameTH = &basicInfo.LastnameTH
			user.FirstNameEN = &basicInfo.FirstnameEN
			user.LastNameEN = &basicInfo.LastnameEN
		}

		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(map[string]interface{}{
			"token": tokenString,
			"user":  user,
		}))
	}
}

func ReserveNotLogin(dbConn *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body LoginDTO
		if err := c.Bind(&body); err != nil || body.FirstName == "" || body.LastName == "" || body.Topic == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid firstname or lastname or topic"})
			return
		}
		tokenString, err := generateJWTToken(body, true)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JWT token"})
			return
		}

		var topic models.Topic
		topicQuery := `SELECT * FROM topics WHERE id = $1`
		err = dbConn.QueryRow(topicQuery, body.Topic).Scan(&topic.ID, &topic.TopicTH, &topic.TopicEN, &topic.Code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve topic"})
			return
		}

		var lastQueueNo string
		query := `SELECT no FROM queues WHERE topic_id = $1 ORDER BY created_at DESC LIMIT 1`
		err = dbConn.QueryRow(query, body.Topic).Scan(&lastQueueNo)
		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve the last queue number"})
			return
		}

		var newQueueNo string
		if lastQueueNo != "" {
			var numPart int
			_, err := fmt.Sscanf(lastQueueNo, topic.Code+"%03d", &numPart)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse the last queue number"})
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

		insertQuery := `INSERT INTO queues (no, firstName, lastName, topic_id, note) 
						VALUES ($1, $2, $3, $4, $5) RETURNING id`
		var queueID int
		err = dbConn.QueryRow(insertQuery, newQueueNo, body.FirstName, body.LastName, body.Topic, note).Scan(&queueID)
		if err != nil {
			log.Printf("Error inserting queue: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create queue"})
			return
		}

		var lastInProgressQueueNo string
		inProgressQuery := `
			SELECT no 
			FROM queues 
			WHERE topic_id = $1 
			AND status = 'IN_PROGRESS' 
			ORDER BY created_at DESC 
			LIMIT 1
		`
		err = dbConn.QueryRow(inProgressQuery, body.Topic).Scan(&lastInProgressQueueNo)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("Error retrieving the last 'IN_PROGRESS' queue: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve the last 'IN_PROGRESS' queue"})
			return
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
			err = dbConn.QueryRow(countWaitingQuery, body.Topic, lastInProgressQueueNo, queueID, topic.Code+"%").Scan(&countWaitingAfterInProgress)
		} else {
			countWaitingQuery = `
				SELECT COUNT(*) 
				FROM queues 
				WHERE topic_id = $1 
				AND status = 'WAITING'
				AND id != $2
				AND no LIKE $3
			`
			err = dbConn.QueryRow(countWaitingQuery, body.Topic, queueID, topic.Code+"%").Scan(&countWaitingAfterInProgress)
		}
		if err != nil {
			log.Printf("Error counting waiting queues after the 'IN_PROGRESS' queue: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count waiting queues after the 'IN_PROGRESS' queue"})
			return
		}

		var queue models.Queue
		queueQuery := `SELECT * FROM queues q WHERE id = $1`
		err = dbConn.QueryRow(queueQuery, queueID).Scan(&queue.ID, &queue.No, &queue.StudentID, &queue.Firstname, &queue.Lastname, &queue.TopicID, &queue.Note, &queue.Status, &queue.CounterID, &queue.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve queue details"})
			return
		}
		queue.Topic = topic

		c.JSON(http.StatusOK, helpers.FormatSuccessResponse(map[string]interface{}{
			"token":   tokenString,
			"queue":   queue,
			"waiting": countWaitingAfterInProgress,
		}))
	}
}
