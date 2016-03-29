package main

import (
	"net/http"
	"io"
	"go/hServer"
)

func serve(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Hello there bitch")
}

func main() {
	http.HandleFunc("/",serve)
	hServer.ListenAndServe(":8000",nil)
}
