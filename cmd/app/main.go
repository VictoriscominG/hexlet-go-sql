package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"database/sql"

	_ "modernc.org/sqlite"

	"example.com/go-sql/internal/storage"
)

func initDB(ctx context.Context, db *sql.DB) error {
	createUsers := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email VARCHAR(255) NOT NULL UNIQUE,
			name VARCHAR(255),
			age INTEGER,
			created_at DATETIME NOT NULL DEFAULT (datetime('now'))
		);
	`

	createCourses := `
		CREATE TABLE IF NOT EXISTS courses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			price INTEGER NOT NULL DEFAULT 0
		);
	`

	if _, err := db.ExecContext(ctx, createUsers); err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	if _, err := db.ExecContext(ctx, createCourses); err != nil {
		return fmt.Errorf("failed to create courses table: %w", err)
	}

	return nil
}

func parseInt(s string) (*int, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func printJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		log.Fatalf("failed to encode JSON: %v", err)
	}
}

func parseStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func parseID(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("--id is required")
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// strPtr и intPtr нужны только для генерации тестовых данных в bulk-команде
func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func main() {
	action := flag.String("action", "", "action: create|get|list|update|delete|bulk")

	email := flag.String("email", "", "email (for create)")
	name := flag.String("name", "", "name (for create/update)")
	age := flag.String("age", "", "age (for create/update)")

	idStr := flag.String("id", "", "user ID (for get/update/delete)")

	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := sql.Open("sqlite", "file:data.db?_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "database ping failed: %v\n", err)
		os.Exit(1)
	}

	if err := initDB(ctx, db); err != nil {
		fmt.Fprintf(os.Stderr, "database initialization failed: %v\n", err)
		os.Exit(1)
	}

	switch *action {
	case "create":
		if *email == "" {
			fmt.Fprintln(os.Stderr, "error: --email is required for create")
			os.Exit(1)
		}

		namePtr := parseStringPtr(*name)
		agePtr, err := parseInt(*age)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid --age: %v\n", err)
			os.Exit(1)
		}

		dto := storage.CreateUserDTO{
			Email: *email,
			Name:  namePtr,
			Age:   agePtr,
		}
		user, err := storage.CreateUser(ctx, db, dto)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: create user failed: %v\n", err)
			os.Exit(1)
		}
		printJSON(user)

	case "get":
		id, err := parseID(*idStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid --id: %v\n", err)
			os.Exit(1)
		}
		user, err := storage.GetUser(ctx, db, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: get user failed: %v\n", err)
			os.Exit(1)
		}
		printJSON(user)

	case "list":
		users, err := storage.ListUsers(ctx, db)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: list users failed: %v\n", err)
			os.Exit(1)
		}
		printJSON(users)

	case "update":
		id, err := parseID(*idStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid --id: %v\n", err)
			os.Exit(1)
		}

		namePtr := parseStringPtr(*name)
		agePtr, err := parseInt(*age)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid --age: %v\n", err)
			os.Exit(1)
		}

		dto := storage.UpdateUserDTO{
			ID:   id,
			Name: namePtr,
			Age:  agePtr,
		}
		user, err := storage.UpdateUser(ctx, db, dto)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: update user failed: %v\n", err)
			os.Exit(1)
		}
		printJSON(user)

	case "delete":
		id, err := parseID(*idStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid --id: %v\n", err)
			os.Exit(1)
		}
		user, err := storage.DeleteUser(ctx, db, id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: delete user failed: %v\n", err)
			os.Exit(1)
		}
		printJSON(user)

	case "bulk":
		bulkAction(ctx, db)

	default:
		fmt.Fprintln(os.Stderr, "error: unknown action. use --action=create|get|list|update|delete|bulk")
		os.Exit(1)
	}
}

func bulkAction(ctx context.Context, db *sql.DB) {
	// Пример тестовых данных для bulk-upsert
	items := []storage.CreateUserDTO{
		{Email: "user1@example.com", Name: strPtr("User One"), Age: intPtr(25)},
		{Email: "user2@example.com", Name: strPtr("User Two"), Age: intPtr(30)},
		{Email: "user3@example.com", Name: strPtr("User Three"), Age: intPtr(35)},
		{Email: "user4@example.com", Name: strPtr("User Four"), Age: intPtr(40)},
		{Email: "user5@example.com", Name: strPtr("User Five"), Age: intPtr(45)},
	}

	// --- Сценарий 1: Bulk upsert через подготовленное выражение ---
	t0 := time.Now()
	count, err := storage.BulkUpsertUsers(ctx, db, items)
	dur := time.Since(t0)

	if err != nil {
		fmt.Fprintf(os.Stderr, "bulk upsert failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("bulk upsert completed: %d rows in %v\n", count, dur)

	// --- Сценарий 2: Один Stmt + много QueryContext по префиксам ---
	const selectQuery = `
		SELECT id, email, name, age, created_at
		FROM users
		WHERE email LIKE ?
	`
	stmt, err := db.PrepareContext(ctx, selectQuery)
	if err != nil {
		fmt.Fprintf(os.Stderr, "prepare select: %v\n", err)
		os.Exit(1)
	}
	defer stmt.Close() // гарантирует закрытие даже при ошибках

	prefixes := []string{"user", "test"} // разные префиксы для демонстрации

	t1 := time.Now()
	users, totalRows, err := storage.ListUsersByEmailPrefix(ctx, stmt, prefixes)
	durSelect := time.Since(t1)

	if err != nil {
		fmt.Fprintf(os.Stderr, "select by prefixes failed: %v\n", err)
		os.Exit(1)
	}

	printJSON(users)
	fmt.Printf("select by prefixes: %d total rows in %v\n", totalRows, durSelect)
}
