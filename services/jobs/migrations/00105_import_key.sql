-- +goose Up
-- A Workday or sitemap job id is unique only within its company's site, so the board is part of the key.
DROP INDEX ux_jobs_origin_external;
CREATE UNIQUE INDEX ux_jobs_origin_board_external ON jobs (origin, board, external_id) WHERE source = 'import';

-- +goose Down
DROP INDEX ux_jobs_origin_board_external;
CREATE UNIQUE INDEX ux_jobs_origin_external ON jobs (origin, external_id) WHERE source = 'import';
