package main

import (
	"net/http"
	"fmt"
)

type AuctionChannelController struct { Controller }

// Stores
func (c *AuctionChannelController) store(w http.ResponseWriter, r  *http.Request) {
	fmt.Println("Hello :D", r.Body)

	c.publish()
}

// Publishes new auction data to Amazon SQS, this service is responsible
// for being the publisher in the pub/sub model, the Relay server
// is the subscriber which streams the data to the consumer via socket.io
func (c *AuctionChannelController) publish() {
	fmt.Println("Pushing data to queue system")
}