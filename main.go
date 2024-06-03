package main

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "./projects.db")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		path TEXT NOT NULL
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(10 << 20) // 10 MB limit
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Get file from form
	file, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Create a new file
	newFile, err := os.Create("./uploads/" + handler.Filename)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer newFile.Close()

	// Write the file to disk
	_, err = io.Copy(newFile, file)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Save project details to database
	name := r.FormValue("name")
	if name == "" {
		name = handler.Filename
	}

	_, err = db.Exec("INSERT INTO projects (name, path) VALUES (?, ?)", name, "./uploads/"+handler.Filename)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "File uploaded successfully: %s", handler.Filename)
}

func viewProjectHandler(w http.ResponseWriter, r *http.Request) {
	projectName := r.URL.Path[len("/view/"):]
	var path string
	err := db.QueryRow("SELECT path FROM projects WHERE name=?", projectName).Scan(&path)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	file, err := os.Open(path)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func main() {
	initDB()
	http.HandleFunc("/upload", uploadFileHandler)
	http.HandleFunc("/view/", viewProjectHandler)
	fmt.Println("Server started at :8081")
	http.ListenAndServe(":8081", nil)
}
