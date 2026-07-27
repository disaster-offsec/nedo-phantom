package api

import (
    "net/http"
    "nedo-phantom/internal/c2/db"
)

func StartAPI(tasks *db.TaskQueue) {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("C2 API"))
    })
    go http.ListenAndServe(":8080", nil)
}
