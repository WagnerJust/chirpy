-- +goose Up
SELECT 'up SQL query';
CREATE TABLE IF NOT EXISTS chirps (
    id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    body TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users ON DELETE CASCADE,
    PRIMARY KEY(id)
);
-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS chirps;
