package main

import (
	"fmt"
	"strings"
)

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

	if a.seller != "" && len(a.items) > 0 {
		id := a.GetPlayer()
		LogInDebugMode("Player: " + strings.Title(a.seller) + " has an id of: " + fmt.Sprint(id))

		// Get the items
		itemsQuery := "SELECT id FROM items " +
			"WHERE name IN ("

		var params []string
		var prices []float32
		for _, item := range a.items {
			itemsQuery += "?,"
			params = append(params, item.name)
			prices = append(prices, item.price)
		}
		itemsQuery = itemsQuery[0:len(itemsQuery)-1] + ")"

		// convert []string to []interface for query
		convertedParams := make([]interface{}, len(params))
		for i, v := range params {
			convertedParams[i] = v
		}

		rows := DB.Query(itemsQuery, convertedParams...)
		var itemIds []int64
		if rows != nil {
			var itemId int64
			defer rows.Close()

			for rows.Next() {
				err := rows.Scan(&itemId)
				if err != nil {
					fmt.Println("Scan error: ", err)
				} else {
					itemIds = append(itemIds, itemId)
				}
			}
		}
		fmt.Println("Inserting auction with ids: ", itemIds)

		auctionQuery := "INSERT INTO auctions (player_id, item_id, price) " +
			" VALUES "

		var auctionParams []interface{}
		for i, itemId := range itemIds {
			auctionQuery += "(?, ?, ?),"
			auctionParams = append(auctionParams, id)
			auctionParams = append(auctionParams, itemId)
			auctionParams = append(auctionParams, prices[i])
		}
		auctionQuery = auctionQuery[0:len(auctionQuery)-1]
		DB.Insert(auctionQuery, auctionParams...)
	} else {
		fmt.Println("Can't save this auction, it does not have a player name or it has no items: ", a)
	}

}

// Attempt to create the player, if they already exist then we select them from the DB
// and return the inserted or selected id
func (a *Auction) GetPlayer() int64 {
	playerQuery := "INSERT IGNORE INTO players (name) VALUES (?)"
	id, err := DB.Insert(playerQuery, a.seller)
	if err != nil {
		fmt.Println("Error inserting player: ", err, id)
	}
	// The player exists, lets create him now
	if err == nil && id == 0 {
		LogInDebugMode("Player already exists with id: " + fmt.Sprint(id))
		playerQuery = "SELECT id FROM players WHERE name = ?"
		rows := DB.Query(playerQuery, strings.Title(a.seller))
		if rows != nil {
			defer rows.Close()

			for rows.Next() {
				err := rows.Scan(&id)
				if err != nil {
					fmt.Println("Scan error: ", err)
				}
			}
		}
	} else if id > 0 {
		LogInDebugMode("Created player with id: ", id)
	}

	return id
}