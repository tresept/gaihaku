package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

// mainPageHandlerは認証後のメインページを表示します
func mainPageHandler(c echo.Context) error {
	// セッションを取得
	sess, err := session.Get("session", c)
	if err != nil {
		// セッションの取得に失敗したら、ログインページへリダイレクト
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}

	// セッションに "authenticated" の値がない、または false の場合はログインページへリダイレクト
	if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}

	studentID := sess.Values["studentID"].(string)
	log.Printf("User %s accessed the main page", studentID)

	// データベースから欠食・外泊記録を取得
	records, err := getGaihakuKesshokuRecords(db, studentID)
	if err != nil {
		log.Printf("Failed to get records for studentID %s: %v", studentID, err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve records.")
	}

	return c.Render(http.StatusOK, "main.html", map[string]interface{}{
		"studentID": studentID,
		"records":   records,
	})
}

func gaihakuHandler(c echo.Context) error {
	// セッションを取得
	sess, err := session.Get("session", c)
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}

	// 認証されていない場合はログインページへリダイレクト
	if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}

	studentID := sess.Values["studentID"].(string)
	log.Printf("User %s accessed the gaihaku page", studentID)

	formValues, err := c.FormParams()
	if err != nil {
		log.Printf("Failed to parse form data: %v", err)
		return c.String(http.StatusBadRequest, "Invalid form data.")
	}

	// 現在の日付から7日分のデータを処理
	now := time.Now()
	for i := 0; i < 7; i++ {
		recordDate := now.AddDate(0, 0, i)
		dateStr := recordDate.Format("2006-01-02")

		// フォームデータから各項目を取得
		// "on"が送信された場合はtrue、そうでない場合はfalseになる
		breakfast := formValues.Get("breakfast-"+dateStr) == "on"
		lunch := formValues.Get("lunch-"+dateStr) == "on"
		dinner := formValues.Get("dinner-"+dateStr) == "on"
		overnight := formValues.Get("overnight-"+dateStr) == "on"
		note := formValues.Get("note-" + dateStr)

		// データベースにデータを挿入または更新
		query := `INSERT INTO gaihaku_kesshoku_records (student_id, record_date, breakfast, lunch, dinner, overnight, note) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (student_id, record_date) DO UPDATE SET breakfast = EXCLUDED.breakfast, lunch = EXCLUDED.lunch, dinner = EXCLUDED.dinner, overnight = EXCLUDED.overnight, note = EXCLUDED.note;`

		_, err = db.Exec(query, studentID, recordDate, !breakfast, !lunch, !dinner, overnight, note)
		if err != nil {
			log.Printf("Failed to insert or update record for %s: %v", dateStr, err)
			return c.String(http.StatusInternalServerError, "Failed to submit record.")
		}
	}

	log.Printf("Successfully submitted records for student %s", studentID)

	// 成功したらメインページにリダイレクト
	return c.Redirect(http.StatusSeeOther, "/main")
}

// loginFormHandlerはログインフォームを表示します
func loginFormHandler(c echo.Context) error {
	return c.Render(http.StatusOK, "login.html", map[string]interface{}{})
}

// loginHandlerはログイン認証処理を行います
func loginHandler(c echo.Context) error {
	studentID := c.FormValue("student_id")
	password := c.FormValue("password")

	if AuthenticateUser(db, studentID, password) {
		// 認証成功
		sess, _ := session.Get("session", c)
		sess.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7,
			HttpOnly: true,
		}
		sess.Values["authenticated"] = true
		sess.Values["studentID"] = studentID

		// セッションの保存をリダイレクトの前に実行
		if err := sess.Save(c.Request(), c.Response()); err != nil {
			log.Printf("Failed to save session: %v", err)
			return c.String(http.StatusInternalServerError, "Failed to login.")
		}

		return c.Redirect(http.StatusSeeOther, "/main")
	}

	// 認証失敗
	return c.Render(http.StatusUnauthorized, "login.html", map[string]interface{}{
		"error": "認証に失敗しました。学籍番号またはパスワードが間違っています。",
	})
}

func logoutHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		// セッションが取得できない場合は、既にログアウト状態とみなす
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}

	// セッションの値をクリアして、有効期限を過去にする
	sess.Options.MaxAge = -1
	if err = sess.Save(c.Request(), c.Response()); err != nil {
		log.Printf("Failed to save session: %v", err)
		return c.String(http.StatusInternalServerError, "Failed to log out.")
	}

	// ログインページにリダイレクト
	return c.Redirect(http.StatusSeeOther, "/")
}
