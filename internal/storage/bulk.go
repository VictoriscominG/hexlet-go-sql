package storage

import (
	"context"
	"database/sql"
	"fmt"
)

// BulkUpsertUsers загружает пользователей через подготовленное выражение.
// Использует INSERT OR REPLACE для эмуляции upsert в SQLite.
func BulkUpsertUsers(ctx context.Context, db *sql.DB, items []CreateUserDTO) (int, error) {
	const query = `
		INSERT OR REPLACE INTO users(email, name, age)
		VALUES(?, ?, ?)
	`

	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("prepare upsert users: %w", err)
	}
	defer stmt.Close() // гарантирует закрытие даже при ошибках

	count := 0
	for _, dto := range items {
		var nameVal interface{} = nil
		if dto.Name != nil {
			nameVal = *dto.Name
		}

		var ageVal interface{} = nil
		if dto.Age != nil {
			ageVal = *dto.Age
		}

		_, err := stmt.ExecContext(ctx, dto.Email, nameVal, ageVal)
		if err != nil {
			// Можно решить: прервать весь bulk или пропустить одну строку.
			// Здесь прерываем, чтобы не портить целостность.
			return count, fmt.Errorf("exec upsert for %s: %w", dto.Email, err)
		}
		count++
	}

	return count, nil
}

// ListUsersByEmailPrefix демонстрирует работу: один Stmt + много QueryContext.
// Это полезно, если нужно делать много выборок с разными параметрами.
func ListUsersByEmailPrefix(ctx context.Context, stmt *sql.Stmt, prefixes []string) ([]User, int, error) {
	var out []User
	totalRows := 0

	for _, prefix := range prefixes {
		rows, err := stmt.QueryContext(ctx, prefix+"%")
		if err != nil {
			return nil, totalRows, fmt.Errorf("query for prefix %q: %w", prefix, err)
		}
		defer rows.Close() // важно: закрываем сразу после каждой итерации

		for rows.Next() {
			var u User
			err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Age, &u.CreatedAt)
			if err != nil {
				return nil, totalRows, fmt.Errorf("scan row: %w", err)
			}
			out = append(out, u)
			totalRows++
		}

		if err := rows.Err(); err != nil {
			return nil, totalRows, fmt.Errorf("rows err for prefix %q: %w", prefix, err)
		}
	}

	return out, totalRows, nil
}
