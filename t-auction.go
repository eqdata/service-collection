package main

import "time"

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
	auctioned_at time.Time
}
