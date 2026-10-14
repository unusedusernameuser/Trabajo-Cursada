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

CREATE TABLE folders (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL 
        REFERENCES users(id) ON DELETE CASCADE,
    folder_name VARCHAR(63) NOT NULL
        CHECK (length(trim(folder_name)) > 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, folder_name)
);

CREATE TABLE tags (
    id SERIAL PRIMARY key,
    user_id INTEGER NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,
    tag_name VARCHAR(63) NOT NULL
        CHECK (length(trim(tag_name)) > 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, tag_name)
);

CREATE TABLE groups (
    id SERIAL PRIMARY KEY,
    owner_id INTEGER NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,
    group_name VARCHAR(63) NOT NULL
        CHECK (length(trim(group_name)) > 0),
    group_description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE groups_members (
    group_id INTEGER NOT NULL
        REFERENCES groups(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, user_id)
);

CREATE TABLE tasks (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL
        REFERENCES users(id) ON DELETE CASCADE,
    folder_id INTEGER
        REFERENCES folders(id) ON DELETE SET NULL,
    group_id INTEGER
        REFERENCES groups(id) ON DELETE CASCADE,
    assigned_to INTEGER
        REFERENCES users(id) on DELETE set NULL,
    title VARCHAR(127) NOT NULL
        CHECK (length(trim(title)) > 0),
    task_description TEXT,
    due_date TIMESTAMP WITH TIME ZONE,
    task_status VARCHAR(15) NOT NULL DEFAULT 'pendiente'
        CHECK (task_status IN ('pendiente', 'en_progreso', 'completada')),
    task_priority VARCHAR(15) NOT NULL DEFAULT('media')
        CHECK (task_priority IN ('baja', 'media', 'alta')),
    attachment_path VARCHAR(255)
        CHECK (attachment_path IS NULL OR length(trim(attachment_path)) > 0),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT task_completed_at_check
        CHECK ((task_status = 'completada') = (completed_at IS NOT NULL))
);

CREATE TABLE tasks_tags (
    task_id INTEGER NOT NULL
        REFERENCES tasks(id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL
        REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY key (task_id, tag_id)
);