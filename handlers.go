package main

import (
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

// loginFormHandlerはログインフォームを表示します
func loginFormHandler(c echo.Context) error {
	return c.Render(http.StatusOK, "login.html", map[string]interface{}{})
}

// loginHandlerはログイン認証処理を行います
func loginHandler(c echo.Context) error {
	studentID := c.FormValue("student_id")
	password := c.FormValue("password")

	if AuthenticateUser(db, studentID, password) {
		sess, _ := session.Get("session", c)
		sess.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7,
			HttpOnly: true,
		}
		sess.Values["authenticated"] = true
		sess.Values["studentID"] = studentID

		if err := sess.Save(c.Request(), c.Response()); err != nil {
			log.Printf("Failed to save session: %v", err)
			return c.String(http.StatusInternalServerError, "Failed to login.")
		}
		return c.Redirect(http.StatusSeeOther, "/main")
	}

	return c.Render(http.StatusUnauthorized, "login.html", map[string]interface{}{
		"error": "認証に失敗しました。学籍番号またはパスワードが間違っています。",
	})
}

// mainPageHandlerは認証後のメインページを表示します
func mainPageHandler(c echo.Context) error {
	// 認証チェック
	sess, err := session.Get("session", c)
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}
	if auth, ok := sess.Values["authenticated"].(bool); !ok || !auth {
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}
	studentID := sess.Values["studentID"].(string)

	// ここに欠食・外泊記録を取得する処理を記述します。
	// まだこの処理は実装していないので、
	// 一旦空のスライスを返してエラーが出ないようにします。
	records := []GaihakuRecord{}

	// テンプレートにデータを渡してレンダリング
	return c.Render(http.StatusOK, "main.html", map[string]interface{}{
		"studentID": studentID,
		"records":   records,
	})
}

// gaihakuRecordHandlerは欠食・外泊届を処理します
func gaihakuRecordHandler(c echo.Context) error {
	// ここに欠食・外泊の登録処理を実装します
	return c.String(http.StatusOK, "Gaihaku registration logic will go here.")
}

// logoutHandlerはセッションを破棄してログアウト処理を行います
func logoutHandler(c echo.Context) error {
	sess, err := session.Get("session", c)
	if err != nil {
		return c.Redirect(http.StatusTemporaryRedirect, "/")
	}
	sess.Values = map[interface{}]interface{}{}
	sess.Options.MaxAge = -1

	if err = sess.Save(c.Request(), c.Response()); err != nil {
		log.Printf("Failed to save session: %v", err)
		return c.String(http.StatusInternalServerError, "Failed to log out.")
	}
	return c.Redirect(http.StatusSeeOther, "/")
}
