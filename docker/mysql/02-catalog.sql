CREATE TABLE books (
 id VARCHAR(36) PRIMARY KEY,
 title VARCHAR(200) NOT NULL,
 genre VARCHAR(30) NOT NULL,
 author_id VARCHAR(36) NOT NULL,
 FOREIGN KEY (author_id) REFERENCES users(id)
);
INSERT INTO books VALUES ('1','Computing Notes','TECHNOLOGY','1'), ('2','Compiler Adventures','TECHNOLOGY','2'), ('3','Analytical Dreams','FICTION','1');
