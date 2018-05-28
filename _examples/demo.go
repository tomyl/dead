package main

import (
	"log"
	"net/http"

	"github.com/tomyl/dead"
)

func main() {
	dead.Default().Watch(".", "templates", "server/*").Main()
	log.Fatal(http.ListenAndServe(":8080", nil))
}
