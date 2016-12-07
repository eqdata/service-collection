package main

import (
	"net/http"
	"fmt"
	"github.com/gorilla/mux"
)

type ItemController struct { Controller }

func (i *ItemController) fetch(w http.ResponseWriter, r  *http.Request) {
	fmt.Println("Fetching item: ", mux.Vars(r)["item_name"])
}