-- name: CreateUser :one
INSERT INTO users (email, name, age)
VALUES (?, ?, ?)
RETURNING id, email, name, age, created_at;

-- name: GetUser :one
SELECT id, email, name, age, created_at
FROM users
WHERE id = ?;

-- name: GetUserByEmail :one
SELECT id, email, name, age, created_at
FROM users
WHERE email = ?;

-- name: ListUsers :many
SELECT id, email, name, age, created_at
FROM users
ORDER BY id;

-- name: UpdateUser :one
UPDATE users
SET name = COALESCE(?, name),
    age  = COALESCE(?, age)
WHERE id = ?
RETURNING id, email, name, age, created_at;

-- name: DeleteUser :one
DELETE FROM users
WHERE id = ?
RETURNING id, email, name, age, created_at;
