-- +goose Up
ALTER TABLE exercises ADD COLUMN description TEXT;
ALTER TABLE exercises ADD COLUMN video_url TEXT;
ALTER TABLE exercises ADD COLUMN img_url TEXT;
ALTER TABLE exercises ADD COLUMN type TEXT NOT NULL DEFAULT 'other';

UPDATE exercises SET type = 'other';

-- +goose Down
ALTER TABLE exercises DROP COLUMN type;
ALTER TABLE exercises DROP COLUMN img_url;
ALTER TABLE exercises DROP COLUMN video_url;
ALTER TABLE exercises DROP COLUMN description;