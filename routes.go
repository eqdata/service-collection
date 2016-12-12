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
		"Store Auction",
		"POST",
		"/channels/auction",
		AC.store,
	},
}