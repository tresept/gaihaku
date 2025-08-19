package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

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

// getAllUsers は全てのユーザー情報を取得します（パスワードを除く）
func getAllUsers(db *sql.DB) ([]User, error) {
	rows, err := db.Query("SELECT id, username, role FROM users ORDER BY id ASC")
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role); err != nil {
			log.Printf("Failed to scan user: %v", err)
			continue
		}
		users = append(users, u)
	}

	return users, nil
}

// createUsersTable はユーザーテーブルを作成します
func createUsersTable(db *sql.DB) error {
	const createTableSQL = `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		role VARCHAR(10) NOT NULL DEFAULT 'user'
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
func AuthenticateUser(db *sql.DB, studentID, password string) (bool, string) {
	var hashedPassword, role string
	query := "SELECT password, role FROM users WHERE username = $1"
	err := db.QueryRow(query, studentID).Scan(&hashedPassword, &role)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, "" // ユーザーが存在しない
		}
		log.Printf("Error during authentication query: %v", err)
		return false, ""
	}

	if bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil {
		return true, role // 認証成功
	}

	return false, "" // パスワードが一致しない
}

// getGaihakuRecords は学生の欠食・外泊記録を取得します
func getGaihakuKesshokuRecords(db *sql.DB, studentID string) ([]GaihakuKesshokuRecord, error) {
	records := []GaihakuKesshokuRecord{}
	// 現在の日付から1週間後までを取得
	rows, err := db.Query(`
	SELECT record_date, breakfast, lunch, dinner, overnight, note 
	FROM gaihaku_kesshoku_records 
	WHERE student_id = $1 AND record_date >= CURRENT_DATE AND record_date <= CURRENT_DATE + INTERVAL '7 days' 
	ORDER BY record_date ASC`, studentID)

	if err != nil {
		return nil, fmt.Errorf("failed to query gaihaku records: %w", err)
	}
	defer rows.Close()

	existingRecords := make(map[string]GaihakuKesshokuRecord)
	now := time.Now()

	for rows.Next() {
		var r GaihakuKesshokuRecord
		if err := rows.Scan(&r.RecordDate, &r.Breakfast, &r.Lunch, &r.Dinner, &r.Overnight, &r.Note); err != nil {
			log.Printf("Failed to scan gaihaku record: %v", err)
			continue
		}
		existingRecords[r.RecordDate.Format("2006-01-02")] = r
	}

	// 今日から7日分のレコードを準備する
	for i := 0; i < 7; i++ {
		recordDate := now.AddDate(0, 0, i)
		dateStr := recordDate.Format("2006-01-02")

		if r, ok := existingRecords[dateStr]; ok {
			// 既存のデータがあればそれを使用
			records = append(records, r)
		} else {
			// なければデフォルト値を使用
			records = append(records, GaihakuKesshokuRecord{
				StudentID:  studentID,
				RecordDate: recordDate,
				Breakfast:  true,
				Lunch:      true,
				Dinner:     true,
				Overnight:  false,
			})
		}
	}

	return records, nil
}

// createAdminUserIfNotExists は、管理者ユーザーが存在しない場合に作成します
func createAdminUserIfNotExists(db *sql.DB) error {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = 'admin')").Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if admin user exists: %w", err)
	}

	if !exists {
		// パスワードをハッシュ化
		password := "admin" // デフォルトのパスワード
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash admin password: %w", err)
		}

		// 管理者ユーザーを挿入
		query := "INSERT INTO users (username, password, role) VALUES ($1, $2, $3)"
		_, err = db.Exec(query, "admin", string(hashedPassword), "admin")
		if err != nil {
			return fmt.Errorf("failed to insert admin user: %w", err)
		}
		log.Println("Default admin user created with username 'admin' and password 'admin'")
	}

	return nil
}
