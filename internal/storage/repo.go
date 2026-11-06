package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// CreateCourse создаёт курс и возвращает его ID
func CreateCourse(ctx context.Context, db *sql.DB, c Course) (int64, error) {
	const query = `
		INSERT INTO courses(slug, title, price)
		VALUES(?, ?, ?)
		RETURNING id
	`

	var id int64
	err := db.QueryRowContext(ctx, query, c.Slug, c.Title, c.Price).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create course: %w", err)
	}
	return id, nil
}

// ListCourses — пагинация + безопасная сортировка по белому списку
func ListCourses(ctx context.Context, db *sql.DB, limit, offset int, order string) ([]Course, error) {
	allowedOrder := map[string]string{
		"price_asc":  "price ASC",
		"price_desc": "price DESC",
		"title_asc":  "title ASC",
		"title_desc": "title DESC",
	}

	ord, ok := allowedOrder[order]
	if !ok {
		ord = "id ASC"
	}

	// Конкатенируем только заранее разрешённое значение сортировки.
	// LIMIT и OFFSET передаём через плейсхолдеры.
	query := `
		SELECT id, slug, title, price
		FROM courses
		ORDER BY ` + ord + `
		LIMIT ? OFFSET ?
	`

	rows, err := db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list courses: %w", err)
	}
	defer rows.Close()

	var out []Course
	for rows.Next() {
		var c Course
		if err := rows.Scan(&c.ID, &c.Slug, &c.Title, &c.Price); err != nil {
			return nil, fmt.Errorf("scan course: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// FindCoursesByIDs — поиск по списку ID с динамическими плейсхолдерами
func FindCoursesByIDs(ctx context.Context, db *sql.DB, ids []int64) ([]Course, error) {
	if len(ids) == 0 {
		return []Course{}, nil
	}

	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1] // убираем последний запятую

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	query := `
		SELECT id, slug, title, price
		FROM courses
		WHERE id IN (` + placeholders + `)
	`

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("find courses: %w", err)
	}
	defer rows.Close()

	var out []Course
	for rows.Next() {
		var c Course
		if err := rows.Scan(&c.ID, &c.Slug, &c.Title, &c.Price); err != nil {
			return nil, fmt.Errorf("scan course by ID: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
