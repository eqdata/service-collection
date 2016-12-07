package main

import (
	"net/http"
	"fmt"
)

type BaseController struct { Controller }

func (b *BaseController) index(w http.ResponseWriter, r  *http.Request) {
	fmt.Println("Route view")
}
