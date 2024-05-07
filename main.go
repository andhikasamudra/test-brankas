package main

import (
	"database/sql"
	"github.com/joho/godotenv"
	"github.com/labstack/gommon/log"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
)

type ImageMetadata struct {
	ID          int
	ContentType string
	Size        int64
	UploadedAt  time.Time
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	e := echo.New()

	e.GET("/", showUploadForm)
	e.POST("/upload", uploadImage)

	e.Start(":8080")
}

// Handler to serve the upload form
func showUploadForm(c echo.Context) error {
	form := `<form action="/upload" method="post" enctype="multipart/form-data">
				<input type="hidden" name="auth" value="123qwe">
				<input type="file" name="data">
				<input type="submit" value="Upload">
			</form>`
	return c.HTML(http.StatusOK, form)
}

func uploadImage(c echo.Context) error {
	auth := c.FormValue("auth")

	if auth != os.Getenv("AUTH_TOKEN") {
		return c.String(http.StatusForbidden, "Forbidden")
	}

	file, err := c.FormFile("data")
	if err != nil {
		return err
	}

	if !isImage(file) {
		return c.String(http.StatusForbidden, "Only images are allowed")
	}

	if file.Size > 8*1024*1024 {
		return c.String(http.StatusForbidden, "Image size exceeds 8MB limit")
	}

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(filepath.Join("uploads", file.Filename))
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	db, err := sql.Open("sqlite3", "images.db")
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS images (id INTEGER PRIMARY KEY, content_type TEXT, size INTEGER, uploaded_at DATETIME)")
	if err != nil {
		return err
	}

	_, err = db.Exec("INSERT INTO images (content_type, size, uploaded_at) VALUES (?, ?, ?)", file.Header.Get("Content-Type"), file.Size, time.Now())
	if err != nil {
		return err
	}

	return c.String(http.StatusOK, "Image uploaded successfully")
}

func isImage(file *multipart.FileHeader) bool {
	imageTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"image/gif":  true,
	}

	f, err := file.Open()
	if err != nil {
		return false
	}
	defer f.Close()

	buffer := make([]byte, 512)
	_, err = f.Read(buffer)
	if err != nil {
		return false
	}

	_, err = f.Seek(0, io.SeekStart)
	if err != nil {
		return false
	}

	contentType := http.DetectContentType(buffer)
	return imageTypes[contentType]
}
