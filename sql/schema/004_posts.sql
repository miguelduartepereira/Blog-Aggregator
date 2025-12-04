-- +goose Up
CREATE TABLE posts (
    id UUID PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    published_at TEXT NOT NULL,
    feed_id UUID NOT NULL REFERENCES feeds(id),
    FOREIGN KEY (feed_id) REFERENCES feeds(id)
);

-- +goose Down
DROP Table posts;