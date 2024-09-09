CREATE TABLE users
(
    uuid UUID PRIMARY KEY,
    login VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL
);

CREATE TABLE refresh_tokens
(
    user_uuid UUID PRIMARY KEY REFERENCES users (uuid),
    token VARCHAR(255) NOT NULL
);

CREATE TABLE outbox_chats
(
    id SERIAL PRIMARY KEY,
    chat UUID NOT NULL,
    author UUID NOT NULL,
    read_only BOOLEAN NOT NULL,
    sent_to_kafka TIMESTAMP
);