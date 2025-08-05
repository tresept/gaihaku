package main

import (
	"log"
	"net/http"

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

	return c.String(http.StatusOK, "Welcome to the main page! You are logged in.")
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
