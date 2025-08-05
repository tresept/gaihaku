package main

import (
	"html/template"
	"io"
	"log"
	"os"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

// TemplateRenderer はHTMLテンプレートをレンダリングする構造体です
type TemplateRenderer struct {
	templates *template.Template
}

// Render はTemplateRendererのメソッドで、テンプレートをレンダリングします
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	// データベースに接続
	var err error
	db, err = connectDB()
	if err != nil {
		log.Fatal("Failed to connect to the database:", err)
	}
	defer db.Close()

	// データベーススキーマを初期化
	if err = initDBSchema(db); err != nil {
		log.Fatal("Failed to initialize database schema:", err)
	}

	// Echoインスタンスの作成
	e := echo.New()

	// テンプレートエンジンの設定
	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}
	e.Renderer = renderer

	// セッション管理ミドルウェアの設定
	secretKey := os.Getenv("SESSION_SECRET_KEY")
	if secretKey == "" {
		log.Fatal("SESSION_SECRET_KEY environment variable not set")
	}
	e.Use(session.Middleware(sessions.NewCookieStore([]byte(secretKey))))

	// ルーティングの設定
	e.GET("/", loginFormHandler)
	e.POST("/login", loginHandler)
	e.GET("/main", mainPageHandler)
	e.POST("/gaihaku", gaihakuRecordHandler)
	e.GET("/logout", logoutHandler)

	// サーバーをポート8080で起動
	e.Logger.Fatal(e.Start(":8080"))
}
