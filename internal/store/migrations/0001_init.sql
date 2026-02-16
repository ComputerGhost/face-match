-- +goose Up

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE categories (
    id BIGSERIAL PRIMARY KEY,
    display_name TEXT NOT NULL,
    is_nsfw BOOL NOT NULL DEFAULT false
);

CREATE TABLE people (
    id BIGSERIAL PRIMARY KEY,
    category_id BIGINT NOT NULL REFERENCES categories(id),
    display_name TEXT NOT NULL,
    disambiguation_tag TEXT NOT NULL DEFAULT '',
    is_hidden BOOL NOT NULL DEFAULT false,
    UNIQUE(category_id, display_name, disambiguation_tag)
);

CREATE TABLE images (
    id BIGSERIAL PRIMARY KEY,
    category_id BIGINT NOT NULL, -- denormalized for fast filters
    person_id BIGINT NOT NULL REFERENCES people(id),
    image_hash BIGINT NOT NULL,
    embedding vector(512) NOT NULL,
    UNIQUE(category_id, image_hash)
);
