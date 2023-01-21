package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"sync"
	"text/template"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Account struct {
    Name   string
    Email  string
}

type PageData struct {
    Title string
}

type Activities struct {
    mu sync.Mutex
    db *sql.DB
}

const database string = "./data.db"
const createDB string = `CREATE TABLE IF NOT EXISTS activities (
    id INTEGER NOT NULL PRIMARY KEY,
    name TEXT,
    email TEXT
)
`

func CreateActivities() (*Activities, error) {
    db, err := sql.Open("sqlite3", database)
    if err != nil {
        return nil, errors.New("Unable to open SQLite database")
    }
    if _, err := db.Exec(createDB); err != nil {
        return nil, errors.New("Unable to create database")
    }
    return &Activities{db: db}, nil
}

func (c *Activities) Insert(acc Account) (int, error) {
    res, err := c.db.Exec("INSERT INTO activities VALUES(NULL, ?, ?);", acc.Name, acc.Email)
    if err != nil {
        return 0, errors.New("Error inserting name into database")
    }
    
    id := int64(0)
    if id, err = res.LastInsertId(); err != nil {
        return 0, errors.New("Error getting the last ID")
    }
    return int(id), nil
}


func (c *Activities) Retrieve(s string) (Account, error) {
    row := c.db.QueryRow("SELECT name, email FROM activities WHERE name = ?", s)

    acc := Account{}
    if err := row.Scan(&acc.Name, &acc.Email); err == sql.ErrNoRows {
        return Account{}, errors.New("Error Retrieving values from databse")
    }
    return acc, nil
}

func (c *Activities) RetrieveList(limit int, offset int) ([]Account, error) {
    rows, err := c.db.Query("SELECT name, email FROM activities ORDER BY id DESC LIMIT ? OFFSET ?", limit, offset)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    accs := make([]Account, 0)
    for rows.Next() {
        acc := Account{}
        err = rows.Scan(&acc.Name, &acc.Email)
        if err != nil {
            return nil, err
        }
        accs = append(accs, acc)
    }
    return accs, nil
}

func IndexHandle( 
    w http.ResponseWriter, r *http.Request, templates *template.Template,
    urlValidator *regexp.Regexp, db * Activities,
) {
    if r.URL.Path != "/" {
        http.Error(w, "Something Went Wrong while attempting to open the server", http.StatusInternalServerError)
        return
    }
    switch r.Method {
    case "GET":
        err := templates.ExecuteTemplate(w, "index.tmpl", nil)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
    case "POST":
        log.Println("[Index] Activated the post request")
        log.Println("Form Value: ", r.FormValue("Name"))
        http.Redirect(w, r, "/writer/" + r.FormValue("Name"), http.StatusFound)
        // http.Redirect(w, r, "https://google.com" , http.StatusFound)
    }
}

func WriterHandle( 
    w http.ResponseWriter, r *http.Request, templates *template.Template,
    urlValidator *regexp.Regexp, db * Activities,
) {
    log.Println("Entered the Writer Handler")
    matches := urlValidator.FindStringSubmatch(r.URL.Path)
    if matches[2] == "" {
        log.Println("Error redirecting to root", matches)
        http.Redirect(w, r, "/", http.StatusInternalServerError)
        return
    }
    switch r.Method {
    case "GET":
        err := templates.ExecuteTemplate(w, "edit.tmpl", nil)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    case "POST":
        log.Println("[Writer] Activated the post request")
        acc := Account{Name: r.FormValue("Name"), Email: r.FormValue("Email")}
        _, err := db.Insert(acc)
        if err != nil {
           http.Error(w, "There was an error writing the data", http.StatusInternalServerError)
           return
        }
        http.Redirect(w, r, "/reader/" + acc.Name, http.StatusFound)

        // http.Redirect(w, r, "/archiver", http.StatusOK)
    }
}

func ReaderHandle(
    w http.ResponseWriter, r *http.Request, templates *template.Template,
    urlValidator *regexp.Regexp, db * Activities,
) {
    log.Println("Entered the Reader Handler")
    matches := urlValidator.FindStringSubmatch(r.URL.Path)
    if matches[2] == "" {
       http.Error(w, "There was an error writing the data", http.StatusInternalServerError)
       return
    }

    acc, err := db.Retrieve(matches[2])

    err = templates.ExecuteTemplate(w, "view.tmpl", acc)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

// Handler Exists to Unglobaglize templates and urlValidator
func MakeHandler(
    fn func(http.ResponseWriter, *http.Request, *template.Template, *regexp.Regexp, *Activities),
    templates *template.Template,
    urlValidator *regexp.Regexp,
    database *Activities,
) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        fn(w, r, templates, urlValidator, database)
    }
}

func main() {
    tmpl, err := filepath.Glob("./templates/*.tmpl")
    if err != nil {
        log.Fatal(err)
    }
    db, err := CreateActivities()
    if err != nil {
        log.Fatal(err)
    }
    templates := template.Must(template.ParseFiles(tmpl...))
    urlValidator := regexp.MustCompile("^/(writer|reader|archiver)/([a-zA-Z0-9]+)$")
    mux := http.NewServeMux()
    mux.HandleFunc("/", MakeHandler(IndexHandle, templates, urlValidator, db))
    mux.HandleFunc("/reader/", MakeHandler(ReaderHandle, templates, urlValidator, db))
    mux.HandleFunc("/writer/", MakeHandler(WriterHandle, templates, urlValidator, db))

    s := &http.Server{
        Addr: ":8080",
        Handler: mux,
        ReadTimeout: 10 * time.Second,
        WriteTimeout: 10 * time.Second,
        MaxHeaderBytes: 1 << 20,
    }

    log.Fatal(s.ListenAndServe())
}
