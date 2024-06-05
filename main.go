package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

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

type TestResults struct {
	UnitTest    string
	QualityGate string
	SmokeTest   string
	Endpoint    string
}

func parseFile(path string) (TestResults, error) {
	file, err := os.Open(path)
	if err != nil {
		return TestResults{}, err
	}
	defer file.Close()

	var results TestResults
	scanner := bufio.NewScanner(file)

	qualityGateOkCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "ok  	go-smoke") {
			results.UnitTest = "Ok"
		}
		if strings.HasPrefix(line, "✅") {
			qualityGateOkCount++
		}
		if strings.HasPrefix(line, "✓") {
			results.SmokeTest = "OK"
		}
		if strings.HasPrefix(line, "http") {
			results.Endpoint = line
		}
	}

	if err := scanner.Err(); err != nil {
		return TestResults{}, err
	}

	if qualityGateOkCount > 5 {
		results.QualityGate = "PASS"
	} else {
		results.QualityGate = "FAIL"
	}

	return results, nil
}

func generateHTML(results TestResults) string {
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Project Results</title>
    <link href="https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css" rel="stylesheet">
</head>
<body>
    <div class="container mx-auto mt-10">
        <h1 class="text-2xl font-bold mb-5">Project Test Results</h1>
        <table class="table-auto w-full text-left">
            <thead>
                <tr>
                    <th class="px-4 py-2">Name</th>
                    <th class="px-4 py-2">Unit Test</th>
                    <th class="px-4 py-2">Quality Gate</th>
                    <th class="px-4 py-2">Smoke Test</th>
                    <th class="px-4 py-2">Endpoint</th>
					<th class="px-4 py-2">Result</th>
                </tr>
            </thead>
            <tbody>
                <tr>
                    <td class="border px-4 py-2">jdnielss-go-smoke</td>
                    <td class="border px-4 py-2">%s</td>
                    <td class="border px-4 py-2">%s</td>
                    <td class="border px-4 py-2">%s</td>
                    <td class="border px-4 py-2"><a href="%s" class="text-blue-500">%s</a></td>
					<td class="border px-4 py-2">PASS</td>
                </tr>
            </tbody>
        </table>
    </div>
</body>
</html>`
	return fmt.Sprintf(html, results.UnitTest, results.QualityGate, results.SmokeTest, results.Endpoint, results.Endpoint)
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
		log.Printf("Error querying database for project %s: %v", projectName, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	results, err := parseFile(path)
	if err != nil {
		log.Printf("Error parsing file %s: %v", path, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	html := generateHTML(results)
	w.Header().Set("Content-Type", "text/html")
	_, err = w.Write([]byte(html))
	if err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

func viewResult(w http.ResponseWriter, r *http.Request) {
	projectName := r.URL.Path[len("/view/result/"):]
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
	http.HandleFunc("/view/result/", viewResult)
	fmt.Println("Server started at :8081")
	http.ListenAndServe(":8081", nil)
}
