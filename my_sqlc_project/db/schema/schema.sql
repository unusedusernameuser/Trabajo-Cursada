CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    handle VARCHAR(31) UNIQUE NOT NULL
        CHECK (handle = lower(handle) AND length(trim(handle)) > 0),
    display_name VARCHAR(63) NOT NULL
        CHECK (length(trim(display_name)) > 0),
    email VARCHAR(255) UNIQUE NOT NULL
        CHECK (email = lower(email) AND length(trim(email)) > 0),
    password_hash VARCHAR(255) NOT NULL
        CHECK (length(trim(password_hash)) > 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);