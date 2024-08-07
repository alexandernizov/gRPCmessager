CREATE TABLE users
(
    uuid uuid PRIMARY KEY,
    login VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL
);

CREATE TABLE refresh_tokens
(
    user_uuid uuid PRIMARY KEY REFERENCES users (uuid),
    token VARCHAR(255) NOT NULL
);