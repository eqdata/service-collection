package main

import "net/http"

type Route struct {
	name 	string
	method 	string
	pattern string
	handler http.HandlerFunc
}

type Routes []Route

// Define any application routes here
var routes = Routes {
	Route {
		"Index",
		"GET",
		"/",
		BC.index,
	},
	Route {
		"Auction",
		"POST",
		"/channel/auction",
		AC.store,
	},
}