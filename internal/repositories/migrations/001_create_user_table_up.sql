CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    passwordHash TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    is_verified BOOLEAN DEFAULT FALSE,
    verify_token VARCHAR(255)
);