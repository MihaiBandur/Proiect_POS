CREATE TABLE users (
    uid       BINARY(16)   PRIMARY KEY,
    username  VARCHAR(255) NOT NULL UNIQUE,
    password  VARCHAR(255) NULL,
    role      ENUM('administrator', 'maintainer') NOT NULL,
    is_active BOOLEAN      NOT NULL DEFAULT FALSE
);

INSERT INTO users (uid, username, password, role, is_active)
VALUES (UNHEX(REPLACE(UUID_v7(), '-', '')), 'admin',
        '$2b$12$sUhmmBo0dgSfgcngYXYpuu2eQ0/Q3/dn0HD5jTaQ1A8MMKeoMDa1a', 'administrator', TRUE);
