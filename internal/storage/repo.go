package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// CreateUser создаёт пользователя и возвращает полную запись.
func CreateUser(ctx context.Context, db *sql.DB, dto CreateUserDTO) (User, error) {
	const query = `
		INSERT INTO users(email, name, age)
		VALUES(?, ?, ?)
		RETURNING id, email, name, age, created_at
	`

	var out User
	// Если Name/Age не переданы — в SQL будет NULL, это ок.
	if err := db.QueryRowContext(ctx, query, dto.Email, dto.Name, dto.Age).
		Scan(&out.ID, &out.Email, &out.Name, &out.Age, &out.CreatedAt); err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return out, nil
}

// GetUser получает пользователя по ID.
func GetUser(ctx context.Context, db *sql.DB, id int64) (User, error) {
	const query = `
		SELECT id, email, name, age, created_at
		FROM users
		WHERE id = ?
	`

	var u User
	if err := db.QueryRowContext(ctx, query, id).
		Scan(&u.ID, &u.Email, &u.Name, &u.Age, &u.CreatedAt); err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

// ListUsers возвращает всех пользователей.
func ListUsers(ctx context.Context, db *sql.DB) ([]User, error) {
	const query = `
		SELECT id, email, name, age, created_at
		FROM users
		ORDER BY id
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var out []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Age, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// UpdateUser делает частичное обновление: обновляет только переданные поля.
func UpdateUser(ctx context.Context, db *sql.DB, dto UpdateUserDTO) (User, error) {
	const query = `
		UPDATE users
		SET name = COALESCE(?, name),
		    age  = COALESCE(?, age)
		WHERE id = ?
		RETURNING id, email, name, age, created_at
	`

	var out User
	if err := db.QueryRowContext(ctx, query, dto.Name, dto.Age, dto.ID).
		Scan(&out.ID, &out.Email, &out.Name, &out.Age, &out.CreatedAt); err != nil {
		return User{}, fmt.Errorf("update user: %w", err)
	}
	return out, nil
}

// DeleteUser удаляет пользователя по ID и возвращает удалённый объект (или пустой с ID).
func DeleteUser(ctx context.Context, db *sql.DB, id int64) (User, error) {
	const getQuery = `
		SELECT id, email, name, age, created_at
		FROM users
		WHERE id = ?
	`
	const deleteQuery = `
		DELETE FROM users
		WHERE id = ?
	`

	// Сначала читаем, чтобы вернуть данные (как будто «получили перед удалением»)
	var out User
	if err := db.QueryRowContext(ctx, getQuery, id).
		Scan(&out.ID, &out.Email, &out.Name, &out.Age, &out.CreatedAt); err != nil {
		// Если не найден — это тоже ошибка, но можно трактовать как «not found»
		return User{}, fmt.Errorf("delete user: get before delete failed: %w", err)
	}

	_, err := db.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return User{}, fmt.Errorf("delete user: exec delete failed: %w", err)
	}

	return out, nil
}
