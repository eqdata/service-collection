package main

import (
	"net/http"
	"log"
	"fmt"
)

func main() {
	fmt.Println("Starting webserver...")
	router := CreateRouter()
	log.Fatal(http.ListenAndServe(":8080", router))
}