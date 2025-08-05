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

	// 新しいユーザーを登録 (初回実行時のみ有効)
	studentID := "20251234"
	password := "my-secret-password"
	// この部分は、ユーザーが既に存在する場合はエラーになるので、コメントアウトするか、別の学籍番号を使ってください
	// err = RegisterUser(db, studentID, password)
	// if err != nil {
	// 	log.Fatal("Failed to register user:", err)
	// }
	// fmt.Printf("User '%s' registered successfully!\n", studentID)

	// ログイン認証を試みる
	fmt.Println("---")
	fmt.Println("Attempting to authenticate user...")

	// 正しいパスワードで認証
	isAuthenticated := AuthenticateUser(db, studentID, password)
	if isAuthenticated {
		fmt.Printf("Authentication successful for user '%s'!\n", studentID)
	} else {
		fmt.Printf("Authentication failed for user '%s'.\n", studentID)
	}

	// 間違ったパスワードで認証
	fmt.Println("---")
	fmt.Println("Attempting to authenticate with wrong password...")
	wrongPassword := "wrong-password"
	isAuthenticated = AuthenticateUser(db, studentID, wrongPassword)
	if isAuthenticated {
		fmt.Printf("Authentication successful for user '%s'!\n", studentID)
	} else {
		fmt.Printf("Authentication failed for user '%s'.\n", studentID)
	}
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
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	query := "INSERT INTO users (username, password) VALUES ($1, $2)"
	_, err = db.Exec(query, studentID, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

// AuthenticateUser はユーザーのログイン認証を行います
func AuthenticateUser(db *sql.DB, studentID, password string) bool {
	// 1. データベースからユーザーのハッシュ化されたパスワードを取得する
	var hashedPassword string
	query := "SELECT password FROM users WHERE username = $1"
	err := db.QueryRow(query, studentID).Scan(&hashedPassword)

	if err != nil {
		// ユーザーが見つからない場合など、エラーが発生したら認証失敗
		log.Printf("Failed to find user '%s': %v\n", studentID, err)
		return false
	}

	// 2. 入力されたパスワードと、データベースから取得したハッシュ化されたパスワードを比較する
	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))

	if err != nil {
		// パスワードが一致しない場合
		log.Println("Invalid password.")
		return false
	}

	// パスワードが一致した場合、認証成功
	return true
}
