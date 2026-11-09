CREATE TABLE users (
    uid       BINARY(16)   PRIMARY KEY,
    username  VARCHAR(255) NOT NULL UNIQUE,
    password  VARCHAR(255) NULL,
    role      ENUM('administrator', 'maintainer') NOT NULL,
    is_active BOOLEAN      NOT NULL DEFAULT FALSE
);
