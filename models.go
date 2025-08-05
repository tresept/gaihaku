package main

import "time"

// User はユーザー情報を格納する構造体です
type User struct {
	ID       int
	Username string
	Password string
}

// GaihakuRecord は欠食・外泊記録のデータ構造を定義します
type GaihakuRecord struct {
	ID         int
	StudentID  string
	RecordDate time.Time
	Breakfast  bool
	Lunch      bool
	Dinner     bool
	Overnight  bool
	Note       string
	CreatedAt  time.Time
}
