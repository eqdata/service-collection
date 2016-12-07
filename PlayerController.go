package main

import (
	"net/http"
	"fmt"
	"github.com/gorilla/mux"
)

type PlayerController struct { Controller }

func (p *PlayerController) fetch(w http.ResponseWriter, r  *http.Request) {
	fmt.Println("Fetching player: ", mux.Vars(r)["player_name"])
}
