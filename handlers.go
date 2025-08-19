package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

// AdminMiddleware は管理ユーザーであるかを確認するミドルウェアです
func AdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		sess, err := session.Get("session", c)
		if err != nil {
			return c.Redirect(http.StatusTemporaryRedirect, "/") // セッション取得失敗
		}

		auth, ok := sess.Values["authenticated"].(bool)
		if !ok || !auth {
			return c.Redirect(http.StatusTemporaryRedirect, "/") // 未認証
		}

		role, ok := sess.Values["role"].(string)
		if !ok || role != "admin" {
			// ここでは管理者でない場合の明確なエラーページへリダイレクトするか、
			// もしくは単にメインページへリダイレクトするなど、仕様に応じた対応を検討
			return c.Redirect(http.StatusTemporaryRedirect, "/main") // 管理者ではないのでメインページへ
		}

		return next(c)
	}
}

// adminDashboardHandler は管理者ダッシュボードを表示します
func adminDashboardHandler(c echo.Context) error {
	users, err := getAllUsers(db)
	if err != nil {
		log.Printf("Failed to get all users: %v", err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve user data.")
	}

	// セッションからフラッシュメッセージを取得
	sess, _ := session.Get("session", c)
	flashes := sess.Flashes("success_message")
	successMessage := ""
	if len(flashes) > 0 {
		successMessage = flashes[0].(string)
	}
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Printf("Failed to save session after reading flash: %v", err)
	}

	return c.Render(http.StatusOK, "admin.html", map[string]interface{}{
		"users":          users,
		"successMessage": successMessage,
	})
}

// adminAddUserFormHandler は新規ユーザー追加フォームを表示します
func adminAddUserFormHandler(c echo.Context) error {
	// エラーメッセージをセッションから取得
	sess, _ := session.Get("session", c)
	flashes := sess.Flashes("error_message")
	errorMessage := ""
	if len(flashes) > 0 {
		errorMessage = flashes[0].(string)
	}
	sess.Save(c.Request(), c.Response())

	return c.Render(http.StatusOK, "admin_add_user.html", map[string]interface{}{
		"errorMessage": errorMessage,
	})
}

// adminAddUserHandler は新規ユーザー追加処理を行います
func adminAddUserHandler(c echo.Context) error {
	studentID := c.FormValue("student_id")
	password := c.FormValue("password")

	if studentID == "" || password == "" {
		sess, _ := session.Get("session", c)
		sess.AddFlash("学籍番号とパスワードは必須です。", "error_message")
		sess.Save(c.Request(), c.Response())
		return c.Redirect(http.StatusSeeOther, "/admin/add_user")
	}

	err := RegisterUser(db, studentID, password)
	if err != nil {
		log.Printf("Failed to register new user by admin: %v", err)
		sess, _ := session.Get("session", c)
		// A more specific error message would be better.
		// For now, a generic one.
		sess.AddFlash("ユーザーの追加に失敗しました。ユーザーが既に存在する可能性があります。", "error_message")
		sess.Save(c.Request(), c.Response())
		return c.Redirect(http.StatusSeeOther, "/admin/add_user")
	}

	// Add a success flash message
	sess, _ := session.Get("session", c)
	message := fmt.Sprintf("ユーザー '%s' を追加しました。", studentID)
	sess.AddFlash(message, "success_message")
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Printf("Failed to save session with flash message: %v", err)
	}

	return c.Redirect(http.StatusSeeOther, "/admin")
}

// adminUpdateUserRecordsHandler は管理者によるユーザーの外泊・欠食記録の更新を処理します
func adminUpdateUserRecordsHandler(c echo.Context) error {
	studentID := c.Param("student_id")
	if studentID == "" {
		return c.String(http.StatusBadRequest, "Student ID is required.")
	}

	formValues, err := c.FormParams()
	if err != nil {
		log.Printf("Failed to parse form data for admin update: %v", err)
		return c.String(http.StatusBadRequest, "Invalid form data.")
	}

	now := time.Now()
	for i := 0; i < 7; i++ {
		recordDate := now.AddDate(0, 0, i)
		dateStr := recordDate.Format("2006-01-02")

		// HTMLのフォーム値からbool値を決定するロジック
		// 食事: 'on' は「欠食」を意味するので、DBでは false
		// 外泊: 'on' は「外泊」を意味するので、DBでは true
		breakfast := formValues.Get("breakfast-"+dateStr) != "on"
		lunch := formValues.Get("lunch-"+dateStr) != "on"
		dinner := formValues.Get("dinner-"+dateStr) != "on"
		overnight := formValues.Get("overnight-"+dateStr) == "on"
		note := formValues.Get("note-" + dateStr)

		query := `INSERT INTO gaihaku_kesshoku_records (student_id, record_date, breakfast, lunch, dinner, overnight, note) VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (student_id, record_date) DO UPDATE SET breakfast = EXCLUDED.breakfast, lunch = EXCLUDED.lunch, dinner = EXCLUDED.dinner, overnight = EXCLUDED.overnight, note = EXCLUDED.note;`

		_, err = db.Exec(query, studentID, recordDate, breakfast, lunch, dinner, overnight, note)
		if err != nil {
			log.Printf("Failed to insert or update record for %s by admin: %v", dateStr, err)
			return c.String(http.StatusInternalServerError, "Failed to submit record.")
		}
	}

	// 成功のフラッシュメッセージを追加
	sess, _ := session.Get("session", c)
	sess.AddFlash("ユーザーの記録を更新しました。", "update_success")
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Printf("Failed to save session with flash message: %v", err)
	}

	// 編集ページにリダイレクト
	return c.Redirect(http.StatusSeeOther, "/admin/user/"+studentID)
}

// adminViewUserRecordsHandler は特定のユーザーの外泊・欠食記録を表示・編集するページです
func adminViewUserRecordsHandler(c echo.Context) error {
	studentID := c.Param("student_id")
	if studentID == "" {
		return c.String(http.StatusBadRequest, "Student ID is required.")
	}

	records, err := getGaihakuKesshokuRecords(db, studentID)
	if err != nil {
		log.Printf("Failed to get records for studentID %s by admin: %v", studentID, err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve records.")
	}

	// 成功メッセージをセッションから取得
	sess, _ := session.Get("session", c)
	flashes := sess.Flashes("update_success")
	successMessage := ""
	if len(flashes) > 0 {
		successMessage = flashes[0].(string)
	}
	sess.Save(c.Request(), c.Response())

	return c.Render(http.StatusOK, "admin_user_records.html", map[string]interface{}{
		"studentID":      studentID,
		"records":        records,
		"successMessage": successMessage,
	})
}

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

	flashes := sess.Flashes("success_message")
	successMessage := ""
	if len(flashes) > 0 {
		successMessage = flashes[0].(string)
	}

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Printf("Failed to save session: %v", err)
	}

	// データベースから欠食・外泊記録を取得
	records, err := getGaihakuKesshokuRecords(db, studentID)
	if err != nil {
		log.Printf("Failed to get records for studentID %s: %v", studentID, err)
		return c.String(http.StatusInternalServerError, "Failed to retrieve records.")
	}

	return c.Render(http.StatusOK, "main.html", map[string]interface{}{
		"studentID":      studentID,
		"records":        records,
		"successMessage": successMessage,
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

	formValues, err := c.FormParams()
	if err != nil {
		log.Printf("Failed to parse form data: %v", err)
		return c.String(http.StatusBadRequest, "Invalid form data.")
	}

	now := time.Now()
	for i := 0; i < 7; i++ {
		recordDate := now.AddDate(0, 0, i)
		dateStr := recordDate.Format("2006-01-02")

		// フォームデータから各項目を取得
		// HTMLのaria-pressed="true"に対応
		breakfast := formValues.Get("breakfast-"+dateStr) == "on"
		lunch := formValues.Get("lunch-"+dateStr) == "on"
		dinner := formValues.Get("dinner-"+dateStr) == "on"
		overnight := formValues.Get("overnight-"+dateStr) == "on"
		note := formValues.Get("note-" + dateStr)

		query := `INSERT INTO gaihaku_kesshoku_records (student_id, record_date, breakfast, lunch, dinner, overnight, note) VALUES ($1, $2, $3, $4, $5, $6, $7)
        ON CONFLICT (student_id, record_date) DO UPDATE SET breakfast = EXCLUDED.breakfast, lunch = EXCLUDED.lunch, dinner = EXCLUDED.dinner, overnight = EXCLUDED.overnight, note = EXCLUDED.note;`

		_, err = db.Exec(query, studentID, recordDate, breakfast, lunch, dinner, overnight, note)
		if err != nil {
			log.Printf("Failed to insert or update record for %s: %v", dateStr, err)
			return c.String(http.StatusInternalServerError, "Failed to submit record.")
		}
	}

	// 成功したらセッションにフラッシュメッセージを保存
	timestamp := time.Now().Format("[15:04]")
	message := fmt.Sprintf("%s 登録を受け付けました。", timestamp)

	sess.AddFlash(message, "success_message")
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Printf("Failed to save session: %v", err)
	}

	// 成功したらメインページにリダイレクト
	return c.Redirect(http.StatusSeeOther, "/main")
}

// loginFormHandlerはログインフォームを表示します
func loginFormHandler(c echo.Context) error {
	sess, _ := session.Get("session", c)

	// セッションに "authenticated" の値があり、trueの場合は /main にリダイレクト
	if auth, ok := sess.Values["authenticated"].(bool); ok && auth {
		// リダイレクトする前に、セッションを保存
		if err := sess.Save(c.Request(), c.Response()); err != nil {
			log.Printf("Failed to save session before redirect: %v", err)
		}
		return c.Redirect(http.StatusSeeOther, "/main")
	}

	// セッションからフラッシュメッセージを取得
	flashes := sess.Flashes("login_error")
	errorMessage := ""
	if len(flashes) > 0 {
		errorMessage = flashes[0].(string)
	}

	// セッションを保存して、メッセージを削除
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Printf("Failed to save session after flashing: %v", err)
	}

	// テンプレートにエラーメッセージを渡してレンダリング
	return c.Render(http.StatusOK, "login.html", map[string]interface{}{
		"error": errorMessage,
	})
}

// loginHandlerはログイン認証処理を行います
func loginHandler(c echo.Context) error {
	studentID := c.FormValue("student_id")
	password := c.FormValue("password")

	authenticated, role := AuthenticateUser(db, studentID, password)
	if authenticated {
		// 認証成功
		sess, _ := session.Get("session", c)
		sess.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7, // 7 days
			HttpOnly: true,
		}
		sess.Values["authenticated"] = true
		sess.Values["studentID"] = studentID
		sess.Values["role"] = role // ロールをセッションに保存

		if err := sess.Save(c.Request(), c.Response()); err != nil {
			log.Printf("Failed to save session: %v", err)
			return c.String(http.StatusInternalServerError, "Failed to login.")
		}

		// 役割に基づいてリダイレクト先を変更
		if role == "admin" {
			return c.Redirect(http.StatusSeeOther, "/admin")
		}
		return c.Redirect(http.StatusSeeOther, "/main")
	}

	// 認証失敗
	sess, _ := session.Get("session", c)
	sess.AddFlash("認証に失敗しました。学籍番号またはパスワードが間違っています。", "login_error")
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		log.Printf("Failed to save session with flash message: %v", err)
	}

	return c.Redirect(http.StatusSeeOther, "/")
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
