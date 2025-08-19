package main

import "time"

type User struct {
	ID       int
	Username string
	Password string
	Role     string
}

type GaihakuKesshokuRecord struct {
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
