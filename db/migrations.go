package db

import (
	"database/sql"
	"fmt"
	"log"
)

func CreateTables(db *sql.DB) {
	createTableQueries := []string{
		`CREATE TABLE IF NOT EXISTS subscriptions (
			firstname VARCHAR(100) NOT NULL,
			lastname VARCHAR(100) NOT NULL,
			endpoint TEXT NOT NULL,
			auth TEXT NOT NULL,
			p256dh TEXT NOT NULL,
			PRIMARY KEY (firstName, lastName)
		);`,
		`CREATE TABLE IF NOT EXISTS counters (
			id SERIAL PRIMARY KEY,
			counter CHAR(1) UNIQUE NOT NULL,
			status BOOLEAN DEFAULT false NOT NULL,
			time_closed TIME DEFAULT '16:00:00' NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			firstName_th VARCHAR(100),
			lastName_th VARCHAR(100),
			firstName_en VARCHAR(100),
			lastName_en VARCHAR(100),
			email VARCHAR(100) UNIQUE NOT NULL,
			counter_id INT REFERENCES counters(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS topics (
			id SERIAL PRIMARY KEY,
			topic_th VARCHAR(255) UNIQUE NOT NULL,
			topic_en VARCHAR(255) UNIQUE NOT NULL,
			code CHAR(1) UNIQUE NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS counter_topics (
			counter_id INT NOT NULL REFERENCES counters(id) ON DELETE CASCADE,
			topic_id INT NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
			PRIMARY KEY (counter_id, topic_id)
		);`,
		`CREATE TABLE IF NOT EXISTS queues (
			id SERIAL PRIMARY KEY,
			no VARCHAR(100) NOT NULL,
			student_id CHAR(9),
			firstName VARCHAR(100) NOT NULL,
			lastName VARCHAR(100) NOT NULL,
			topic_id INT REFERENCES topics(id) ON DELETE CASCADE ON UPDATE CASCADE,
			note TEXT,
			status VARCHAR(10) DEFAULT 'WAITING' NOT NULL,
			counter_id INT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS feedbacks (
			id SERIAL PRIMARY KEY,
			user_id INT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
			topic VARCHAR(255) NOT NULL,
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
