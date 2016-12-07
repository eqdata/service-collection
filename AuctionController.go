package main

import (
	"net/http"
	"fmt"
	"time"
)

type AuctionController struct { Controller }

// Stores auction data to the Amazon RDS storage once it has been parsed
func (c *AuctionController) store(w http.ResponseWriter, r  *http.Request) {
	fmt.Println("Hello :D", r.Body)


	go c.parse() // We don't care when this finishes so run it as an async go process
}

// Publishes new auction data to Amazon SQS, this service is responsible
// for being the publisher in the pub/sub model, the Relay server
// is the subscriber which streams the data to the consumer via socket.io
func (c *AuctionController) publish() {
	fmt.Println("Pushing data to queue system")
}

//
func (c *AuctionController) parse() {
	// This just emulates that this is now asynchronous
	time.Sleep(2 * time.Second)
	fmt.Println("Parsing the data!")
	c.publish()
}