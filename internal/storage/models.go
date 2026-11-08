package storage

import "time"

// User — сущность (модель БД). Возвращается наружу.
type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      *string   `json:"name,omitempty"` // nullable
	Age       *int      `json:"age,omitempty"`  // nullable
	CreatedAt time.Time `json:"created_at"`
}

// CreateUserDTO — DTO для создания пользователя (принимает CLI).
type CreateUserDTO struct {
	Email string  `json:"email"`
	Name  *string `json:"name,omitempty"`
	Age   *int    `json:"age,omitempty"`
}

// UpdateUserDTO — DTO для частичного обновления.
type UpdateUserDTO struct {
	ID   int64   `json:"id"`
	Name *string `json:"name,omitempty"`
	Age  *int    `json:"age,omitempty"`
}
