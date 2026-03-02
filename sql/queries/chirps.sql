-- name: CreateChirp :one
INSERT INTO chirps (id, created_at, updated_at, body, user_id)
VALUES (
gen_random_uuid(), $1, $2, $3, $4
)
RETURNING *;
