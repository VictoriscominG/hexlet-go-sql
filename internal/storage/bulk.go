package storage

import (
	"context"
	"database/sql"
	"fmt"
)

func BulkUpsertUsers(ctx context.Context, db *sql.DB, items []CreateUserDTO) (int, error) {
	count := 0

	err := withTx(ctx, db, func(tx *sql.Tx) error {
		const query = `INSERT OR REPLACE INTO users(email, name, age) VALUES(?, ?, ?)`

		stmt, stmtErr := tx.PrepareContext(ctx, query)
		if stmtErr != nil {
			return fmt.Errorf("prepare upsert stmt: %w", stmtErr)
		}
		defer stmt.Close()

		for _, dto := range items {
			var nameVal interface{} = nil
			if dto.Name != nil {
				nameVal = *dto.Name
			}

			var ageVal interface{} = nil
			if dto.Age != nil {
				ageVal = *dto.Age
			}

			_, execErr := stmt.ExecContext(ctx, dto.Email, nameVal, ageVal)
			if execErr != nil {
				return fmt.Errorf("exec upsert for %s: %w", dto.Email, execErr)
			}
			count++
		}
		return nil
	})

	return count, err
}

func ListUsersByEmailPrefix(ctx context.Context, stmt *sql.Stmt, prefixes []string) ([]User, int, error) {
	var out []User
	totalRows := 0

	for _, prefix := range prefixes {
		rows, err := stmt.QueryContext(ctx, prefix+"%")
		if err != nil {
			return nil, totalRows, fmt.Errorf("query for prefix %q: %w", prefix, err)
		}
		// Закрываем сразу после обработки этой выборки
		defer func() {
			_ = rows.Close()
		}()

		for rows.Next() {
			var u User
			if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Age, &u.CreatedAt); err != nil {
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
