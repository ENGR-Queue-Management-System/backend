package models

import (
	"time"
)

type Room struct {
	ID   uint   `json:"id"`
	Room string `json:"room"`
}

type Topic struct {
	ID     uint   `json:"id"`
	Topic  string `json:"topic"`
	Code   string `json:"code"`
	Status bool   `json:"status"`
	RoomID uint   `json:"roomId"`
	Room   Room   `json:"room"`
}

type User struct {
	ID          uint    `json:"id"`
	FirstNameTH *string `json:"firstNameTH"`
	LastNameTH  *string `json:"lastNameTH"`
	FirstNameEN *string `json:"firstNameEN"`
	LastNameEN  *string `json:"lastNameEN"`
	Email       string  `json:"email"`
	RoomID      *uint   `json:"roomId"`
	Room        *Room   `json:"room"`
}

type Feedback struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"userId"`
	User      User      `json:"user"`
	TopicID   uint      `json:"topicId"`
	Topic     Topic     `json:"topic"`
	Feedback  string    `json:"feedback"`
	CreatedAt time.Time `json:"createdAt"`
}

type Queue struct {
	ID          uint      `json:"id"`
	No          int       `json:"no"`
	StudentID   string    `json:"studentId"`
	Firstname   string    `json:"firstname"`
	Lastname    string    `json:"lastname"`
	TopicID     uint      `json:"topicId"`
	Topic       Topic     `json:"topic"`
	Description string    `json:"description"`
	Status      bool      `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UserID      *uint     `json:"userId"`
	User        *User     `json:"user"`
}
