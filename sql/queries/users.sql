-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES (
gen_random_uuid(), $1, $2, $3
)
RETURNING *;


-- name: DeleteAllUsers :exec
DELETE FROM users;


-- name: FindUserById :one
SELECT * FROM users WHERE id = $1;
