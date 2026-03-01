-- +goose Up
SELECT 'up SQL query';
CREATE TABLE IF NOT EXISTS users (
    id UUID NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    email TEXT NOT NULL,
    PRIMARY KEY(id)
);

-- +goose Down
SELECT 'down SQL query';
DROP TABLE IF EXISTS users;
