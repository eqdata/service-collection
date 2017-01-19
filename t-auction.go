package main

import (
	"fmt"
	"strings"
	"github.com/bradfitz/gomemcache/memcache"
	"strconv"
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
	Seller string
	Items []Item
}

func (a *Auction) Save() {
	fmt.Println("Saving auction for seller: " + a.Seller + ", with items: ", a.Items)

	if a.Seller != "" && len(a.Items) > 0 {
		playerId := a.GetPlayer()
		LogInDebugMode("Player: " + strings.Title(a.Seller) + " has an id of: " + fmt.Sprint(playerId))

		// Get the items
		itemsQuery := "SELECT id FROM items " +
			"WHERE name IN ("

		var params []string
		var prices []float32
		var quants []int32
		for _, item := range a.Items {
			itemsQuery += "?,"
			params = append(params, item.Name)
			prices = append(prices, item.Price)
			if item.Quantity == 0 {
				item.Quantity = 1
			}
			quants = append(quants, item.Quantity)
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

			for rows.Next() {
				err := rows.Scan(&itemId)
				if err != nil {
					fmt.Println("Scan error: ", err)
				} else {
					itemIds = append(itemIds, itemId)
				}
			}
			if err := rows.Err(); err != nil {
				fmt.Println("ROW ERROR: ", err.Error())
			}
			DB.CloseRows(rows)
		}
		LogInDebugMode("Inserting auction with ids: ", itemIds)

		auctionQuery := "INSERT INTO auctions (player_id, item_id, price, quantity) " +
			" VALUES "

		var auctionParams []interface{}
		for i, itemId := range itemIds {
			if !a.itemRecentlyAuctionedByPlayer(itemId, prices[i], quants[i]) {
				auctionQuery += "(?, ?, ?, ?),"
				auctionParams = append(auctionParams, playerId)
				auctionParams = append(auctionParams, itemId)
				auctionParams = append(auctionParams, prices[i])
				auctionParams = append(auctionParams, quants[i])
			}
		}
		auctionQuery = auctionQuery[0:len(auctionQuery)-1]
		LogInDebugMode("Query is: ", auctionQuery)
		LogInDebugMode("Params are: ", auctionParams)
		if DB.conn != nil && len(auctionParams) > 0 {
			DB.Insert(auctionQuery, auctionParams...)
		}
	} else {
		fmt.Println("Can't save this auction, it does not have a player name or it has no items: ", a)
	}

}

// Check memcached to see whether or not this item was already recently auctioned, if it was
// then we wont save its record out to the DB unless the price has changed
func (a *Auction) itemRecentlyAuctionedByPlayer(itemId int64, price float32, quantity int32) bool {

	var s Sale = Sale{Seller:a.Seller, ItemId: itemId, Price: price, Quantity: quantity}

	// Attempt to fetch the item from memached
	mc := memcache.New(MC_HOST + ":" + MC_PORT)

	// Use an _ as we don't need to use the cache item returned
	key := strings.TrimSpace("sale:" + strconv.FormatInt(itemId, 10) + ":player:" + a.Seller)
	fmt.Println("Key is: ", key)
	mcItem, err := mc.Get(key)
	if err != nil {
		if err.Error() == "memcache: cache miss" {
			fmt.Println("Couldn't find item in the cache")
			fmt.Println("Setting item: " + key + " in cache for: " + fmt.Sprint(SALE_CACHE_TIME_IN_SECS) + " seconds")
			mc.Set(&memcache.Item{Key: fmt.Sprint(key), Value: s.serialize(), Expiration: SALE_CACHE_TIME_IN_SECS})
			return false
		} else {
			fmt.Println("Error was: ", err.Error())
			return false
		}
	} else if mcItem != nil {
		LogInDebugMode("Got item from memcached: ", mcItem)
		var s Sale
		s = s.deserialize(mcItem.Value)

		if s.Price == price {
			fmt.Println("The prices haven't changed so we will not insert for id: ", itemId)
			return true
		}

		fmt.Println("The old price of: " + fmt.Sprint(s.Price) + " is different to: " + fmt.Sprint(price) + " busting the cache!")
		return false
	}

	return false
}

// Attempt to create the player, if they already exist then we select them from the DB
// and return the inserted or selected id
func (a *Auction) GetPlayer() int64 {
	playerQuery := "INSERT IGNORE INTO players (name) VALUES (?)"
	id, err := DB.Insert(playerQuery, a.Seller)
	if err != nil {
		fmt.Println("Error inserting player: ", err, id)
	}
	// The player exists, lets create him now
	if err == nil && id == 0 {
		LogInDebugMode("Player already exists with id: " + fmt.Sprint(id))
		playerQuery = "SELECT id FROM players WHERE name = ?"
		rows := DB.Query(playerQuery, strings.Title(a.Seller))

		if rows != nil {
			for rows.Next() {
				err := rows.Scan(&id)
				if err != nil {
					fmt.Println("Scan error: ", err)
				}
			}
		}
		if err = rows.Err(); err != nil {
			fmt.Println("ROW ERROR: ", err.Error())
		}
		DB.CloseRows(rows)
	} else if id > 0 {
		LogInDebugMode("Created player with id: ", id)
	}

	return id
}