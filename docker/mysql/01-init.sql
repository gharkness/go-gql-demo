CREATE TABLE users (
 id VARCHAR(36) PRIMARY KEY,
 username VARCHAR(80) NOT NULL UNIQUE,
 name VARCHAR(120) NOT NULL
);
INSERT INTO users VALUES ('1', 'ada', 'Ada Lovelace'), ('2', 'grace', 'Grace Hopper');
