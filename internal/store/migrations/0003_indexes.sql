-- +goose Up
CREATE INDEX images_person_id_idx ON images(person_id);
CREATE INDEX images_category_id_idx ON images(category_id);
CREATE INDEX images_embedding_idx ON images USING hnsw (embedding vector_cosine_ops);
