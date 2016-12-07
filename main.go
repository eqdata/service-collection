package main

import (
	"net/http"
	"log"
)

func main() {
	router := CreateRouter()
	log.Fatal(http.ListenAndServe(":8080", router))
}