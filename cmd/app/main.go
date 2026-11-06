package main

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"time"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	ID       int64  `json:"id"`
	NAME     string `json:"name"`
	EMAIL    string `json:"email"`
	PASSWORD string `json:"-"`
}

func main() {
	db, err := sql.Open("sqlite3", "data.db")
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * 60 * time.Second)

	ctx := context.Background()

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL
	);`

	_, err = db.ExecContext(ctx, createTableSQL)
	if err != nil {
		log.Fatalf("failed to create table: %v", err)
	}

	insertSQL := `INSERT INTO users (name, email, password_hash) VALUES (?,?,?);`
	_, err = db.ExecContext(ctx, insertSQL, "Orlando", "Orlan@gmail.com", "2H^sisters2026")
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			log.Println("user already exists (expected for repeated runs)")
		} else {
			log.Fatalf("failed to insert user: %v", err)
		}
	}

	var u User
	row := db.QueryRowContext(ctx, "SELECT id, name, email FROM users WHERE email = ?", "Orlan@gmail.com")
	err = row.Scan(&u.ID, &u.NAME, &u.EMAIL)
	if err != nil {
		log.Fatalf("failed to scan user: %v", err)
	}

	payload, _ := json.MarshalIndent(u, "", "  ")

	log.Printf("Test user loaded: %s", payload)
}
