package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"example.com/go-sql/internal/db"
)

// --- Доменные модели и DTO ---

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      *string   `json:"name,omitempty"` // nullable
	Age       *int      `json:"age,omitempty"`  // nullable
	CreatedAt time.Time `json:"created_at"`
}

type Course struct {
	ID    int64  `json:"id"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Price int    `json:"price"`
}

type CreateUserDTO struct {
	Email string  `json:"email"`
	Name  *string `json:"name,omitempty"`
	Age   *int    `json:"age,omitempty"`
}

type UpdateUserDTO struct {
	ID   int64   `json:"id"`
	Name *string `json:"name,omitempty"`
	Age  *int    `json:"age,omitempty"`
}

type CreateCourseDTO struct {
	Slug  string
	Title string
	Price int
}

// --- Репозиторий: базовые CRUD через сгенерированный sqlc-код ---

type Repo struct {
	conn *sql.DB
	q    *db.Queries
}

func NewRepo(sqlDB *sql.DB) *Repo {
	return &Repo{conn: sqlDB, q: db.New(sqlDB)}
}

func (r *Repo) CreateUser(ctx context.Context, dto CreateUserDTO) (User, error) {
	row, err := r.q.CreateUser(ctx, db.CreateUserParams{
		Email: dto.Email,
		Name:  toNullString(dto.Name),
		Age:   toNullInt64(dto.Age),
	})
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return userFromDB(row), nil
}

func (r *Repo) GetUser(ctx context.Context, id int64) (User, error) {
	row, err := r.q.GetUser(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	return userFromDB(row), nil
}

func (r *Repo) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return User{}, fmt.Errorf("get user by email: %w", err)
	}
	return userFromDB(row), nil
}

func (r *Repo) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := r.q.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	out := make([]User, 0, len(rows))
	for _, row := range rows {
		out = append(out, userFromDB(row))
	}
	return out, nil
}

func (r *Repo) UpdateUser(ctx context.Context, dto UpdateUserDTO) (User, error) {
	row, err := r.q.UpdateUser(ctx, db.UpdateUserParams{
		Name: toNullString(dto.Name), // NULL => COALESCE оставит старое значение
		Age:  toNullInt64(dto.Age),
		ID:   dto.ID,
	})
	if err != nil {
		return User{}, fmt.Errorf("update user: %w", err)
	}
	return userFromDB(row), nil
}

func (r *Repo) DeleteUser(ctx context.Context, id int64) (User, error) {
	row, err := r.q.DeleteUser(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("delete user: %w", err)
	}
	return userFromDB(row), nil
}

// --- Courses ---

func (r *Repo) CreateCourse(ctx context.Context, dto CreateCourseDTO) (Course, error) {
	row, err := r.q.CreateCourse(ctx, db.CreateCourseParams{
		Slug:  dto.Slug,
		Title: dto.Title,
		Price: int64(dto.Price),
	})
	if err != nil {
		return Course{}, fmt.Errorf("create course: %w", err)
	}
	return courseFromDB(row), nil
}

func (r *Repo) GetCourseBySlug(ctx context.Context, slug string) (Course, error) {
	row, err := r.q.GetCourseBySlug(ctx, slug)
	if err != nil {
		return Course{}, fmt.Errorf("get course by slug: %w", err)
	}
	return courseFromDB(row), nil
}

func (r *Repo) ListCourses(ctx context.Context) ([]Course, error) {
	rows, err := r.q.ListCourses(ctx)
	if err != nil {
		return nil, fmt.Errorf("list courses: %w", err)
	}
	out := make([]Course, 0, len(rows))
	for _, row := range rows {
		out = append(out, courseFromDB(row))
	}
	return out, nil
}

func (r *Repo) DeleteCourse(ctx context.Context, slug string) error {
	if err := r.q.DeleteCourse(ctx, slug); err != nil {
		return fmt.Errorf("delete course: %w", err)
	}
	return nil
}

// --- Ручные сценарии: bulk upsert в транзакции и один Stmt на много запросов ---
// sqlc не покрывает такие случаи (динамический bulk, повторное использование
// подготовленного выражения), поэтому они остаются здесь.

func (r *Repo) BulkUpsertUsers(ctx context.Context, items []CreateUserDTO) (int, error) {
	count := 0

	err := r.withTx(ctx, func(tx *sql.Tx) error {
		const query = `INSERT OR REPLACE INTO users(email, name, age) VALUES(?, ?, ?)`

		stmt, stmtErr := tx.PrepareContext(ctx, query)
		if stmtErr != nil {
			return fmt.Errorf("prepare upsert stmt: %w", stmtErr)
		}
		defer stmt.Close() // гарантирует закрытие даже при ошибках

		for _, dto := range items {
			if _, execErr := stmt.ExecContext(ctx, dto.Email, toNullString(dto.Name), toNullInt64(dto.Age)); execErr != nil {
				return fmt.Errorf("exec upsert for %s: %w", dto.Email, execErr)
			}
			count++
		}
		return nil
	})

	return count, err
}

func (r *Repo) ListUsersByEmailPrefix(ctx context.Context, prefixes []string) ([]User, int, error) {
	const selectQuery = `
		SELECT id, email, name, age, created_at
		FROM users
		WHERE email LIKE ?
	`

	stmt, err := r.conn.PrepareContext(ctx, selectQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("prepare select: %w", err)
	}
	defer stmt.Close() // Stmt закрывается один раз, после всех префиксов

	var out []User
	totalRows := 0

	for _, prefix := range prefixes {
		rows, err := stmt.QueryContext(ctx, prefix+"%")
		if err != nil {
			return nil, totalRows, fmt.Errorf("query for prefix %q: %w", prefix, err)
		}

		for rows.Next() {
			var u User
			if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Age, &u.CreatedAt); err != nil {
				_ = rows.Close()
				return nil, totalRows, fmt.Errorf("scan row: %w", err)
			}
			out = append(out, u)
			totalRows++
		}

		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return nil, totalRows, fmt.Errorf("rows err for prefix %q: %w", prefix, err)
		}
		_ = rows.Close() // закрытие сразу после обработки выборки
	}

	return out, totalRows, nil
}

func (r *Repo) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() { _ = tx.Rollback() }()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// --- Конвертация между sql.Null* и указателями ---

func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func fromNullString(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func toNullInt64(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}

func fromNullInt64(ni sql.NullInt64) *int {
	if !ni.Valid {
		return nil
	}
	v := int(ni.Int64)
	return &v
}

func userFromDB(u db.User) User {
	return User{
		ID:        u.ID,
		Email:     u.Email,
		Name:      fromNullString(u.Name),
		Age:       fromNullInt64(u.Age),
		CreatedAt: u.CreatedAt,
	}
}

func courseFromDB(c db.Course) Course {
	return Course{
		ID:    c.ID,
		Slug:  c.Slug,
		Title: c.Title,
		Price: int(c.Price),
	}
}
