package main

// export SESSION_SECRET_KEY="your-secret-key-here"

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// グローバル変数としてデータベース接続を保持
var db *sql.DB

// connectDB はデータベースに接続します
func connectDB() (*sql.DB, error) {
	connStr := "user=user password=password dbname=mydatabase host=db sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}
	fmt.Println("Successfully connected to the database!")

	return db, nil
}

// initDBSchema はデータベーススキーマを初期化します
func initDBSchema(db *sql.DB) error {
	if err := createUsersTable(db); err != nil {
		return err
	}
	log.Println("Users table created or already exists!")

	if err := createGaihakuKesshokuRecordsTable(db); err != nil {
		return err
	}
	log.Println("Gaihaku records table created or already exists!")

	return nil
}

// createUsersTable はユーザーテーブルを作成します
func createUsersTable(db *sql.DB) error {
	const createTableSQL = `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL
	);`
	_, err := db.Exec(createTableSQL)
	return err
}

// createGaihakuKesshokuRecordsTable は欠食・外泊・点呼記録テーブルを作成します
func createGaihakuKesshokuRecordsTable(db *sql.DB) error {
	const createTableSQL = `
	CREATE TABLE IF NOT EXISTS gaihaku_kesshoku_records (
		id SERIAL PRIMARY KEY,
		student_id VARCHAR(50) NOT NULL,
		record_date DATE NOT NULL,
		breakfast BOOLEAN NOT NULL DEFAULT TRUE,
		lunch BOOLEAN NOT NULL DEFAULT TRUE,
		dinner BOOLEAN NOT NULL DEFAULT TRUE,
		overnight BOOLEAN NOT NULL DEFAULT FALSE,
		roll_call BOOLEAN NOT NULL DEFAULT FALSE,
		note TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		UNIQUE (student_id, record_date)
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
	var hashedPassword string
	query := "SELECT password FROM users WHERE username = $1"
	err := db.QueryRow(query, studentID).Scan(&hashedPassword)

	if err != nil {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}

// getGaihakuRecords は学生の欠食・外泊記録を取得します
func getGaihakuRecords(db *sql.DB, studentID string) ([]GaihakuRecord, error) {
	records := []GaihakuRecord{}
	// 現在の日付から1週間後までを取得
	rows, err := db.Query("SELECT record_date, breakfast, lunch, dinner, overnight, note FROM gaihaku_records WHERE student_id = $1 AND record_date >= CURRENT_DATE AND record_date <= CURRENT_DATE + INTERVAL '7 days' ORDER BY record_date ASC", studentID)
	if err != nil {
		return nil, fmt.Errorf("failed to query gaihaku records: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var r GaihakuRecord
		if err := rows.Scan(&r.RecordDate, &r.Breakfast, &r.Lunch, &r.Dinner, &r.Overnight, &r.Note); err != nil {
			log.Printf("Failed to scan gaihaku record: %v", err)
			continue
		}
		records = append(records, r)
	}
	return records, nil
}
