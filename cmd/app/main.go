package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"database/sql"

	_ "modernc.org/sqlite"

	"example.com/go-sql/internal/storage"
)

func initDB(ctx context.Context, db *sql.DB) error {
	createUsers := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL
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

// ensureTestUser вставляет тестового пользователя, если его ещё нет
func ensureTestUser(ctx context.Context, db *sql.DB) error {
	const insertUserSQL = `
		INSERT INTO users (name, email, password_hash)
		VALUES (?, ?, ?)
	`
	_, err := db.ExecContext(ctx, insertUserSQL, "Orlando", "Orlan@gmail.com", "2H^sisters2026")
	if err != nil {
		// Ignore error if the user already exists (UNIQUE constraint)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			log.Println("user already exists (expected for repeated runs)")
			return nil
		}
		return fmt.Errorf("failed to insert test user: %w", err)
	}
	return nil
}

func parseIntList(s string) ([]int64, error) {
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	res := make([]int64, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		v, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, err
		}
		res = append(res, v)
	}
	return res, nil
}

func main() {
	// Флаги для CLI по курсам
	action := flag.String("action", "list", "action: create|list|find")
	slug := flag.String("slug", "", "slug for create")
	title := flag.String("title", "", "title for create")
	price := flag.Int("price", 0, "price for create")

	limit := flag.Int("limit", 10, "limit for list")
	offset := flag.Int("offset", 0, "offset for list")
	order := flag.String("order", "id_asc", "sorting for list: price_asc|price_desc|title_asc|title_desc")

	idsStr := flag.String("ids", "", "comma-separated list of IDs for find")

	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := sql.Open("sqlite", "file:data.db?_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}

	// Создаём таблицы и добавляем тестового пользователя
	if err := initDB(ctx, db); err != nil {
		log.Fatalf("database initialization failed: %v", err)
	}
	if err := ensureTestUser(ctx, db); err != nil {
		log.Fatalf("test user setup failed: %v", err)
	}

	// Выводим тестового пользователя в JSON
	var u struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	row := db.QueryRowContext(ctx, "SELECT id, name, email FROM users WHERE email = ?", "Orlan@gmail.com")
	err = row.Scan(&u.ID, &u.Name, &u.Email)
	if err != nil {
		log.Fatalf("failed to scan user row: %v", err)
	}
	payload, _ := json.MarshalIndent(u, "", "  ")
	log.Printf("Test user loaded: %s", payload)

	// CLI for courses
	switch *action {
	case "create":
		if *slug == "" || *title == "" {
			log.Fatal("the 'create' action requires -slug and -title flags")
		}
		c := storage.Course{
			Slug:  *slug,
			Title: *title,
			Price: *price,
		}
		id, err := storage.CreateCourse(ctx, db, c)
		if err != nil {
			log.Fatalf("course creation failed: %v", err)
		}
		fmt.Printf("Course created with id=%d\n", id)

	case "list":
		courses, err := storage.ListCourses(ctx, db, *limit, *offset, *order)
		if err != nil {
			log.Fatalf("listing courses failed: %v", err)
		}
		if len(courses) == 0 {
			fmt.Println("No courses found.")
			return
		}
		for _, c := range courses {
			fmt.Printf("[%d] %s (%s) — %d RUB\n", c.ID, c.Title, c.Slug, c.Price)
		}

	case "find":
		ids, err := parseIntList(*idsStr)
		if err != nil {
			log.Fatalf("invalid format for -ids: %v", err)
		}
		courses, err := storage.FindCoursesByIDs(ctx, db, ids)
		if err != nil {
			log.Fatalf("searching courses by IDs failed: %v", err)
		}
		if len(courses) == 0 {
			fmt.Println("No courses found for the specified IDs.")
			return
		}
		for _, c := range courses {
			fmt.Printf("[%d] %s (%s) — %d RUB\n", c.ID, c.Title, c.Slug, c.Price)
		}

	default:
		log.Fatalf("unknown action: %s", *action)
	}
}
