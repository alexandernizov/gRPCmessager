CREATE TABLE users
(
    uuid UUID PRIMARY KEY,
    login VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL
);

CREATE TABLE refresh_tokens
(
    user_uuid UUID PRIMARY KEY REFERENCES users (uuid) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL
);

CREATE TABLE chats
(
    uuid UUID PRIMARY KEY,
    owner UUID NOT NULL,
    read_only BOOLEAN NOT NULL,
    dead_line TIMESTAMP NOT NULL
);

CREATE TABLE messages
(   
    id SERIAL PRIMARY KEY,
    chat_uuid UUID NOT NULL REFERENCES chats (uuid) ON DELETE CASCADE,
    author_uuid UUID NOT NULL REFERENCES users (uuid) ON DELETE CASCADE,
    body VARCHAR(255) NOT NULL,
    published TIMESTAMP NOT NULL
);

CREATE TABLE outbox
(   
    uuid UUID PRIMARY KEY,
    topic VARCHAR(255) NOT NULL,
    message VARCHAR(255) NOT NULL,
    sent_at TIMESTAMP
);