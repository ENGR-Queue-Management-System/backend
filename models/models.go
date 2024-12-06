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
	RoomID uint   `json:"room_id"`
	Room   Room   `json:"room"`
}

type User struct {
	ID        uint   `json:"id"`
	StudentID string `json:"student_id"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Email     string `json:"email"`
	RoomID    uint   `json:"room_id"`
	Room      Room   `json:"room"`
}

type Feedback struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	User      User      `json:"user"`
	TopicID   uint      `json:"topic_id"`
	Topic     Topic     `json:"topic"`
	Feedback  string    `json:"feedback"`
	CreatedAt time.Time `json:"created_at"`
}

type Queue struct {
	ID          uint      `json:"id"`
	No          int       `json:"no"`
	StudentID   string    `json:"student_id"`
	Firstname   string    `json:"firstname"`
	Lastname    string    `json:"lastname"`
	TopicID     uint      `json:"topic_id"`
	Topic       Topic     `json:"topic"`
	Description string    `json:"description"`
	Status      bool      `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UserID      uint      `json:"user_id"`
	User        User      `json:"user"`
}
