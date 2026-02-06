package database

import (
    "fmt"
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    config "github.com/kennyniu2/booker/configs"  
)

var DB *sqlx.DB

func InitDB(cfg *config.Config) (*sqlx.DB, error) {
    connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
        
    db, err := sqlx.Connect("postgres", connStr)
    if err != nil {
        return nil, err
    }
    
    DB = db
    
    
    if err := createTables(); err != nil {
        return nil, err
    }
    
    return db, nil
}

func createTables() error {
    
    schema := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        username VARCHAR(100) UNIQUE NOT NULL,
        email VARCHAR(100) UNIQUE NOT NULL,
        password_hash VARCHAR(255) NOT NULL,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
    
    CREATE TABLE IF NOT EXISTS books (
        id SERIAL PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        author VARCHAR(255) NOT NULL,
        description TEXT,
        isbn VARCHAR(20),
        cover_url TEXT,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
    
    CREATE TABLE IF NOT EXISTS user_books (
        id SERIAL PRIMARY KEY,
        user_id INT REFERENCES users(id),
        book_id INT REFERENCES books(id),
        status VARCHAR(20) NOT NULL, -- reading, completed, want-to-read
        rating INT CHECK (rating >= 1 AND rating <= 5),
        review TEXT,
        progress INT DEFAULT 0,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        UNIQUE(user_id, book_id)
    );
    `
    
    _, err := DB.Exec(schema)
    return err
}