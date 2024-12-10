package db

import (
	"database/sql"
	"fmt"
	"log"
)

func CreateTables(db *sql.DB) {
	createTableQueries := []string{
		`CREATE TABLE IF NOT EXISTS rooms (
			id SERIAL PRIMARY KEY,
			room VARCHAR(255) UNIQUE NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			firstName_TH VARCHAR(100),
			lastName_TH VARCHAR(100),
			firstName_EN VARCHAR(100),
			lastName_EN VARCHAR(100),
			email VARCHAR(100) UNIQUE NOT NULL,
			room_id INT REFERENCES rooms(id) ON DELETE SET NULL
		);`,
		`CREATE TABLE IF NOT EXISTS topics (
			id SERIAL PRIMARY KEY,
			topic VARCHAR(255) NOT NULL,
			code CHAR NOT NULL,
			status BOOLEAN DEFAULT false,
			room_id INT REFERENCES rooms(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS queues (
			id SERIAL PRIMARY KEY,
			no VARCHAR(100) NOT NULL,
			studentId CHAR(9) NOT NULL,
			firstName VARCHAR(100) NOT NULL,
			lastName VARCHAR(100) NOT NULL,
			topic_id INT REFERENCES topics(id) NOT NULL,
			description TEXT NOT NULL,
			status BOOLEAN DEFAULT false,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			user_id INT REFERENCES users(id)
		);`,
		`CREATE TABLE IF NOT EXISTS feedbacks (
			id SERIAL PRIMARY KEY,
			user_id INT REFERENCES users(id) NOT NULL,
			topic_id INT REFERENCES topics(id) NOT NULL,
			feedback TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, query := range createTableQueries {
		_, err := db.Exec(query)
		if err != nil {
			log.Fatalf("Failed to execute query: %s\nError: %v", query, err)
		} else {
			fmt.Println("Successfully executed query:", query)
		}
	}
}

func CreateRooms(db *sql.DB) {
	createRoomsQueries := []string{
		`INSERT INTO rooms (room) VALUES ('งานบริการการศึกษา')
			ON CONFLICT (room) DO NOTHING;`,
		`INSERT INTO rooms (room) VALUES ('งานพัฒนาคุณภาพนักศึกษา')
			ON CONFLICT (room) DO NOTHING;`,
	}

	for _, query := range createRoomsQueries {
		_, err := db.Exec(query)
		if err != nil {
			log.Fatalf("Failed to execute query: %s\nError: %v", query, err)
		} else {
			fmt.Println("Successfully executed query:", query)
		}
	}
}

func CreateUsers(db *sql.DB) {
	createUsersQueries := []string{
		`INSERT INTO users (email) 
			VALUES ('thanaporn_chan@cmu.ac.th')
			ON CONFLICT (email) DO NOTHING;`,
		`INSERT INTO users (email) 
			VALUES ('sawit_cha@cmu.ac.th')
			ON CONFLICT (email) DO NOTHING;`,
		`INSERT INTO users (email) 
			VALUES ('worapitcha_muangyot@cmu.ac.th')
			ON CONFLICT (email) DO NOTHING;`,
	}

	for _, query := range createUsersQueries {
		_, err := db.Exec(query)
		if err != nil {
			log.Fatalf("Failed to execute query: %s\nError: %v", query, err)
		} else {
			fmt.Println("Successfully executed query:", query)
		}
	}
}
