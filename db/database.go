package database

import (
    "fmt"
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"
    config "github.com/kennyniu2/bcj/configs"  
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
    
    CREATE TABLE IF NOT EXISTS clubs (
        id SERIAL PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        description TEXT,
        created_by INT REFERENCES users(id),
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
    
    CREATE TABLE IF NOT EXISTS club_members (
        club_id INT REFERENCES clubs(id),
        user_id INT REFERENCES users(id),
        joined_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        is_admin BOOLEAN DEFAULT FALSE,
        PRIMARY KEY (club_id, user_id)
    );
    
    CREATE TABLE IF NOT EXISTS club_books (
        id SERIAL PRIMARY KEY,
        club_id INT REFERENCES clubs(id),
        book_id INT REFERENCES books(id),
        start_date DATE,
        end_date DATE,
        status VARCHAR(20) NOT NULL, -- current, previous, upcoming
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
    
    CREATE TABLE IF NOT EXISTS discussions (
        id SERIAL PRIMARY KEY,
        club_book_id INT REFERENCES club_books(id),
        title VARCHAR(255) NOT NULL,
        content TEXT,
        created_by INT REFERENCES users(id),
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
    
    CREATE TABLE IF NOT EXISTS comments (
        id SERIAL PRIMARY KEY,
        discussion_id INT REFERENCES discussions(id),
        content TEXT NOT NULL,
        created_by INT REFERENCES users(id),
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
    
    CREATE TABLE IF NOT EXISTS votes (
        id SERIAL PRIMARY KEY,
        club_id INT REFERENCES clubs(id),
        title VARCHAR(255) NOT NULL,
        description TEXT,
        start_date TIMESTAMP WITH TIME ZONE,
        end_date TIMESTAMP WITH TIME ZONE,
        created_by INT REFERENCES users(id),
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
    
    CREATE TABLE IF NOT EXISTS vote_options (
        id SERIAL PRIMARY KEY,
        vote_id INT REFERENCES votes(id),
        book_id INT REFERENCES books(id),
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
    );
    
    CREATE TABLE IF NOT EXISTS vote_responses (
        vote_option_id INT REFERENCES vote_options(id),
        user_id INT REFERENCES users(id),
        created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
        PRIMARY KEY (vote_option_id, user_id)
    );
    `
    
    _, err := DB.Exec(schema)
    return err
}