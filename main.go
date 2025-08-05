package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session" // この行があるか確認してください
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// TemplateRenderer はHTMLテンプレートをレンダリングする構造体する
type TemplateRenderer struct {
	// Go標準の html/templateという構造体を使う
	templates *template.Template
}

// Render はTemplateRendererのメソッドで、テンプレートをレンダリングする
// (t *TemplateRenderer) は、TemplateRenderer構造体のメソッドであることを示す
// w.io.Writerで、レンダリング結果を書き込む
// name は、どのテンプレートファイルを使うか（login / main / ...).html
// 要するに、これはテンプレートにデータを埋め込んで返す関数

func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

// グローバル変数としてデータベース接続を保持
var db *sql.DB

func main() {
	// PostgreSQLへの接続情報
	connStr := "user=user password=password dbname=mydatabase host=db sslmode=disable"

	// データベースに接続
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// データベース接続の確認
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

	// Echoインスタンスの作成
	e := echo.New()

	// テンプレートエンジンの設定
	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}
	e.Renderer = renderer

	// セッション管理ミドルウェアの設定
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("your-secret-key"))))

	// ルーティングの設定
	e.GET("/", loginFormHandler) // / にアクセスしたらログインフォームを表示
	e.POST("/login", loginHandler)
	e.GET("/main", mainPageHandler)
	e.GET("/logout", logoutHandler)

	// サーバーをポート8080で起動
	e.Logger.Fatal(e.Start(":8080"))
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
	var hashedPassword string
	query := "SELECT password FROM users WHERE username = $1"
	err := db.QueryRow(query, studentID).Scan(&hashedPassword)

	if err != nil {
		return false
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return false
	}

	return true
}
