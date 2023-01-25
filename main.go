package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
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
    mu          sync.Mutex
    db          *sql.DB
    regex       *regexp.Regexp
    templates   *template.Template
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

func (c *Activities) Delete(s string) (int, error)  {
    res, err := c.db.Exec("DELETE FROM activities WHERE name=?", s)
    if err != nil {
        return 0, errors.New("Error deleting element from database")
    }
     id := int64(0)
     if id, err = res.LastInsertId(); err != nil {
         return 0, errors.New("Error getting the last ID")
     }
     return int(id), nil
}

func IndexHandle(w http.ResponseWriter, r *http.Request, db * Activities) {
    log.Println("Entered the Index Handler")
    if r.URL.Path != "/" {
        http.Error(w, "Something Went Wrong while attempting to open the server", http.StatusInternalServerError)
        return
    }
    switch r.Method {
    case "GET":
        err := db.templates.ExecuteTemplate(w, "index.html", nil)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
        }
    case "POST":
        log.Println("[Index] Activated the post request")
        // log.Println("Form Value: ", r.FormValue("Name"))
        http.Redirect(w, r, "/writer/", http.StatusFound)
        // http.Redirect(w, r, "https://google.com" , http.StatusFound)
    }
}

func WriterHandle(w http.ResponseWriter, r *http.Request, db * Activities) {
    log.Println("Entered the Writer Handler")
    switch r.Method {
    case "GET":
        err := db.templates.ExecuteTemplate(w, "edit.html", nil)
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
        http.Redirect(w, r, "/reader/" + r.FormValue("Name"), http.StatusFound)
    }
}

func ViewsHandle( w http.ResponseWriter, r *http.Request, db * Activities) {
    log.Println("Entered the Views Handler")
    accs, err := db.RetrieveList(100, 0)
    if err != nil { http.Error(w, "There was an error querying the values", http.StatusInternalServerError)}
    switch (r.Method) {
    case "GET":
        err = db.templates.ExecuteTemplate(w, "views.html", accs)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    case "POST":
        log.Println(r.FormValue("submit"))
        _, err = db.Delete(r.FormValue("submit"))
        if err != nil {
            http.Error(w, "There was an error deleting the data", http.StatusInternalServerError)
            return
        }
        http.Redirect(w, r, "/deleted/", http.StatusFound)
    }
}

func ReaderHandle(w http.ResponseWriter, r *http.Request, db * Activities) {
    log.Println("Entered the Reader Handler")
    matches := db.regex.FindStringSubmatch(r.URL.Path)
    // log.Println(matches)
    // if matches[2] == "" {
    //    http.Error(w, "There was an error writing the data", http.StatusInternalServerError)
    //    return
    // }

    if len(matches) < 3 {
        return
    }
    acc, err := db.Retrieve(matches[2])

    switch r.Method {
    case "GET":
        err = db.templates.ExecuteTemplate(w, "view.html", acc)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    case "POST":
        http.Redirect(w, r, "/views/", http.StatusFound)
    default: 
        log.Println("Bruh")
    }
}

func DeletedHandle(w http.ResponseWriter, r *http.Request, db * Activities) {
    log.Println("Entered the Deleted Handler")
    switch r.Method {
    case "GET":
        err := db.templates.ExecuteTemplate(w, "deleted.html", nil)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    case "POST":
        // log.Println(r.Form)
        // log.Println(r.FormValue("return"))
        // log.Println(r.FormValue("view"))
        if r.FormValue("view") == "view" {
            http.Redirect(w, r, "/views/", http.StatusFound)
        } else if r.FormValue("return") == "return" {
            http.Redirect(w, r, "/", http.StatusFound)
        }
    }
}

func MakeHandler(fn func(http.ResponseWriter, *http.Request, *Activities), database *Activities) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        fn(w, r, database)
    }
}

func main() {
    if _, err := os.Stat("./data.db"); errors.Is(err, os.ErrNotExist) {
        os.Create("data.db")
    }
    html, err := filepath.Glob("./website/*.html"); if err != nil {
        log.Fatal(err)
    }
    db, err := CreateActivities(); if err != nil {
        log.Fatal(err)
    }
    db.templates = template.Must(template.ParseFiles(html...))
    db.regex = regexp.MustCompile("^/(writer|reader|archiver|views)/([a-zA-Z0-9]+)$")
    mux := http.NewServeMux()
    mux.HandleFunc("/", MakeHandler(IndexHandle, db))
    mux.HandleFunc("/reader/", MakeHandler(ReaderHandle, db))
    mux.HandleFunc("/writer/", MakeHandler(WriterHandle, db))
    mux.HandleFunc("/views/", MakeHandler(ViewsHandle,  db))
    mux.HandleFunc("/deleted/", MakeHandler(DeletedHandle,  db))
    mux.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("website/css"))))

    s := &http.Server{
        Addr: ":8080",
        Handler: mux,
        ReadTimeout: 10 * time.Second,
        WriteTimeout: 10 * time.Second,
        MaxHeaderBytes: 1 << 20,
    }

    log.Fatal(s.ListenAndServe())
}
