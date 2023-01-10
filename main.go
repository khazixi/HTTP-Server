package main

import (
	"fmt"
	"io"
	"net/http"
)

func welcome(w http.ResponseWriter, r * http.Request) {
    io.WriteString(w, "Welcome!")
}

func main() {
    /*
    f, err := os.ReadFile("./index.html")
    if err != nil {
        log.Fatal(err)
    }
    */
    http.HandleFunc("/", welcome)

    http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprint(w, "This is where the API will be served up eventually, not yet.")
    })

    http.ListenAndServe(":5050", nil)
}

