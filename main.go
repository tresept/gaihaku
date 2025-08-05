package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// PostgreSQLへの接続情報
	connStr := "user=user password=password dbname=mydatabase host=db sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Successfully connected to the database!")

	// ユーザーテーブルを作成
	err = createUsersTable(db)
	if err != nil {
		log.Fatal("Failed to create users table:", err)
	}
	fmt.Println("Users table created or already exists!")

	// 新しいユーザーを登録
	studentID := "20251234"
	password := "my-secret-password"

	err = RegisterUser(db, studentID, password)
	if err != nil {
		log.Fatal("Failed to register user:", err)
	}
	fmt.Printf("User '%s' registered successfully!\n", studentID)
}

// createUsersTable はデータベースにユーザーテーブルを作成します
func createUsersTable(db *sql.DB) error {
	const createTableSQL = `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL
	);`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to execute SQL: %w", err)
	}

	return nil
}

// RegisterUser は新しいユーザーを登録します
func RegisterUser(db *sql.DB, studentID, password string) error {
	// 1. パスワードをハッシュ化する
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// 2. データベースに学籍番号とハッシュ化されたパスワードを挿入する
	query := "INSERT INTO users (username, password) VALUES ($1, $2)"
	_, err = db.Exec(query, studentID, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}
