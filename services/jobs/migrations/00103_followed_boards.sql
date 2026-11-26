-- +goose Up
-- Company job boards people follow from a job they added; the importer reads them with the ones in boards.yaml.
CREATE TABLE followed_boards (
    platform TEXT NOT NULL,
    board TEXT NOT NULL,
    company TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (platform, board)
);

-- +goose Down
DROP TABLE followed_boards;
