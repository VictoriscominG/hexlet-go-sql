-- name: CreateCourse :one
INSERT INTO courses (slug, title, price)
VALUES (?, ?, ?)
RETURNING id, slug, title, price;

-- name: GetCourseBySlug :one
SELECT id, slug, title, price
FROM courses
WHERE slug = ?;

-- name: ListCourses :many
SELECT id, slug, title, price
FROM courses
ORDER BY id;

-- name: DeleteCourse :exec
DELETE FROM courses
WHERE slug = ?;
