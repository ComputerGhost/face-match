-- +goose Up

INSERT INTO categories (display_name, is_nsfw) VALUES
('Actor', false),
('KPop Idol', false),
('Singer', false)
;
