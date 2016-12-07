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
		"Store Auction",
		"POST",
		"/channels/auction",
		AC.store,
	},
	Route {
		"Fetch Item",
		"GET",
		"/items/{item_name}",
		IC.fetch,
	},
	Route {
		"Fetch Player",
		"GET",
		"/players/{player_name}",
		PC.fetch,
	},
}