package main

import "fmt"

/*
 |-------------------------------------------------------------------------
 | Type: Auction
 |--------------------------------------------------------------------------
 |
 | Represent an auction
 |
 | @member seller (string) : The name of the person selling this item
 | @member items ([]Item) : An array of WTS items associated with this specific auction
 | @member auction_at (time.Time) : Timestamp of when this was auctioned
 |
 */

type Auction struct {
	seller string
	items []Item
}

func (a *Auction) Save() {
	fmt.Println("Saving auction for seller: " + a.seller + ", with items: ", a.items)
}