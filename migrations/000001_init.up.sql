CREATE TABLE users
(
    uuid uuid not null unique,
    login varchar(255) not null,
    password varchar(255) not null
);
