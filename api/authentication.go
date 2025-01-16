package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"src/helpers"
	"src/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"gorm.io/gorm"
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
		if v.StudentID != "" {
			claims["studentId"] = v.StudentID
		}
		if notAdmin {
			claims["role"] = helpers.STUDENT
		} else {
			claims["role"] = helpers.ADMIN
		}
		if firstName == "" {
			firstName = helpers.Capitalize(v.FirstnameEN)
		}
		if lastName == "" {
			lastName = helpers.Capitalize(v.LastnameEN)
		}
		claims["faculty"] = v.OrganizationNameTH
	case ReserveDTO:
		firstName = *v.FirstName
		lastName = *v.LastName
		claims["role"] = helpers.STUDENT
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

func Authentication(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body AuthDTO
		if err := c.Bind(&body); err != nil || body.Code == "" || body.RedirectURI == "" {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Invalid authorization code or redirect URI")
			return
		}
		accessToken, err := getEntraIDAccessToken(body.Code, body.RedirectURI)
		if err != nil || accessToken == "" {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Cannot get EntraID access token")
			return
		}
		basicInfo, err := getCMUBasicInfo(accessToken)
		if err != nil || basicInfo == nil {
			helpers.FormatErrorResponse(c, http.StatusBadRequest, "Cannot get CMU basic info")
			return
		}

		var user models.User
		result := db.Where("email = ?", basicInfo.CmuitAccount).First(&user)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			if basicInfo.ItAccountTypeID == STUDENT.String() {
				tokenString, err := generateJWTToken(*basicInfo, true)
				if err != nil {
					helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to generate JWT token")
					return
				}
				helpers.FormatSuccessResponse(c, map[string]interface{}{"token": tokenString})
				return
			} else {
				helpers.FormatErrorResponse(c, http.StatusForbidden, "Cannot access")
				return
			}
		}

		tokenString, err := generateJWTToken(*basicInfo, false)
		if err != nil {
			helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to generate JWT token")
			return
		}

		if user.FirstNameEN == nil || user.LastNameEN == nil {
			user.FirstNameTH = &basicInfo.FirstnameTH
			user.LastNameTH = &basicInfo.LastnameTH
			user.FirstNameEN = &basicInfo.FirstnameEN
			user.LastNameEN = &basicInfo.LastnameEN
			if err := db.Save(&user).Error; err != nil {
				helpers.FormatErrorResponse(c, http.StatusInternalServerError, "Failed to update user data")
				return
			}
		}

		helpers.FormatSuccessResponse(c, map[string]interface{}{
			"token": tokenString,
			"user":  user,
		})
	}
}
