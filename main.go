package main

import (
	"fmt"
	"log"
	"net/http"
    "html/template"
	"os"
)

// Remember This Link: https://golang.google.cn/doc/articles/wiki/

type Page struct {
    Title string
    Body []byte
}

func (p * Page) save() error {
    filename := p.Title + ".txt"
    return os.WriteFile(filename, p.Body, 0600)
}

func loadPage(title string) (*Page, error) {
    filename := title + ".txt"
    body, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    return &Page{Title: title, Body: body}, nil
}

func handler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Welcome to: %s\n", r.URL.Path[1:])
}

func viewHandler(w http.ResponseWriter, r * http.Request) {
    title := r.URL.Path[len("/view/"):]
    p, err := loadPage(title)
    if err != nil {
        p = &Page{Title: "Fuck", Body: []byte("Shit")}
    }
    fmt.Fprintf(w, "<h1>%s</h1><div>%s</div>", p.Title, p.Body)
}

func editHandler(w http.ResponseWriter, r *http.Request) {
    title := r.URL.Path[len("/edit/"):]
    p, err := loadPage(title)
    if err != nil {
        p = &Page{Title: title}
    }
    fmt.Fprintf(
        w, "<h1>Editing %s</h1>" +
        "<form action=\"/save/%s\" method=\"POST\">" +
        "<textarea name=\"body\">%s</textarea>" +
        "<input type=\"submit\" value=\"Save\">" +
        "</form>",
        p.Title, p.Title, p.Body,
    )
    
}

func main() {
    http.HandleFunc("/", handler)
    http.HandleFunc("/view/", viewHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}

