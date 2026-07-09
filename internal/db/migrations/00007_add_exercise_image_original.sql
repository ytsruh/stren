-- +goose Up
-- Stores the storage key of the "original" image variant uploaded
-- alongside the display image. The display image (img_url) is a
-- 800x600 centre-cropped JPEG used in cards/dialogs; this column
-- holds the 1920x1440 source surrogate that the same upload
-- produced. Both columns are populated together by the admin image
-- upload route and cleared together on clear/replace.
ALTER TABLE exercises ADD COLUMN img_url_original TEXT;

-- +goose Down
ALTER TABLE exercises DROP COLUMN img_url_original;
