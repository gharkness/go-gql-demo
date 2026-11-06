CREATE TABLE reviews (
 id VARCHAR(36) PRIMARY KEY,
 book_id VARCHAR(36) NOT NULL,
 author_id VARCHAR(36) NOT NULL,
 body TEXT NOT NULL,
 rating INT NOT NULL CHECK (rating BETWEEN 1 AND 5),
 created_at DATETIME(6) NOT NULL,
 FOREIGN KEY (book_id) REFERENCES books(id),
 FOREIGN KEY (author_id) REFERENCES users(id),
 INDEX (book_id, created_at)
);
INSERT INTO reviews VALUES ('1','1','2','A pioneering read.',5,'2026-01-01 12:00:00');
