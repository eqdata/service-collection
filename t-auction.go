package main

import (
	"fmt"
	"github.com/bradfitz/gomemcache/memcache"
	"strconv"
	"strings"
	"time"
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
	Seller    string
	Timestamp time.Time
	Items     []Item
	Server    string
	itemLine  string
	raw       string
}

func (a *Auction) ExtractQueryInformation(callback func(string, []interface{})) {
	//fmt.Println("Saving auction for seller: " + a.Seller + ", with " + fmt.Sprint(len(a.Items)) + " items.")

	if a.Seller != "" && len(a.Items) > 0 {
		playerId := a.GetPlayer()
		LogInDebugMode("Player: " + strings.Title(a.Seller) + " has an id of: " + fmt.Sprint(playerId))

		// Get the items
		itemsQuery := "SELECT id, displayName FROM items " +
			"WHERE displayName IN ("

		var params []string
		var prices []float32
		var quants []int32
		var sellable []bool
		for _, item := range a.Items {
			itemsQuery += "?,"
			params = append(params, strings.TrimSpace(item.Name))
			prices = append(prices, item.Price)
			sellable = append(sellable, item.selling)
			if item.Quantity == 0 {
				item.Quantity = 1
			}
			quants = append(quants, int32(item.Quantity))
		}
		itemsQuery = itemsQuery[0:len(itemsQuery)-1] + ")" // remove the last ','

		// convert []string to []interface for query
		convertedParams := make([]interface{}, len(params))
		for i, v := range params {
			convertedParams[i] = v
		}

		//fmt.Println(convertedParams)
		//fmt.Println(itemsQuery)

		rows := DB.Query(itemsQuery, convertedParams...)
		if rows != nil {
			var itemId int64
			var name string

			for rows.Next() {
				err := rows.Scan(&itemId, &name)
				if err != nil {
					fmt.Println("Scan error: ", err)
				} else {
					for i, item := range a.Items {
						//fmt.Println("Checking if: " + item.Name + " is equal to: " + name)
						if strings.ToLower(strings.TrimSpace(item.Name)) == strings.ToLower(name) {
							a.Items[i].id = itemId
							//fmt.Println("Item: " + item.Name + " is equal to: " + name + " setting id to: " + fmt.Sprint(itemId))
						}
					}
				}
			}
			if err := rows.Err(); err != nil {
				fmt.Println("ROW ERROR: ", err.Error())
			}
			DB.CloseRows(rows)
		}

		auctionQuery := ""

		var auctionParams []interface{}
		for i, item := range a.Items {
			LogInDebugMode("Checking item: ", item.Name+" for seller: "+a.Seller)
			if !a.itemRecentlyAuctionedByPlayer(item.id, prices[i], quants[i]) && item.id > 0 {
				auctionQuery += "(?, ?, ?, ?, ?, ?, ?),"
				auctionParams = append(auctionParams, playerId)
				auctionParams = append(auctionParams, item.id)
				auctionParams = append(auctionParams, prices[i])
				auctionParams = append(auctionParams, quants[i])
				auctionParams = append(auctionParams, a.Server)
				//auctionParams = append(auctionParams, a.Timestamp)
				auctionParams = append(auctionParams, (a.Seller + " auctions, '" + a.itemLine + "'"))
				auctionParams = append(auctionParams, sellable[i])
			} else if item.id <= 0 {
				LogInDebugMode("Item: ", item.Name+" does not have an id :(")
			} else {
				LogInDebugMode("Item: ", item.Name+" was recently sold")
			}
		}

		callback(auctionQuery, auctionParams)

	} else {
		LogInDebugMode("Can't save this auction, it does not have a player name or it has no items: ", a)
		callback("", nil)
	}

}

// Check memcached to see whether or not this item was already recently auctioned, if it was
// then we wont save its record out to the DB unless the price has changed
func (a *Auction) itemRecentlyAuctionedByPlayer(itemId int64, price float32, quantity int32) bool {

	var s Sale = Sale{Seller: a.Seller, ItemId: itemId, Price: price, Quantity: quantity}

	// Attempt to fetch the item from memached
	mc := memcache.New(MC_HOST + ":" + MC_PORT)

	// Use an _ as we don't need to use the cache item returned
	key := strings.TrimSpace("server:" + a.Server + ":sale:" + strconv.FormatInt(itemId, 10) + ":player:" + a.Seller)
	//fmt.Println("Key is: ", key)
	mcItem, err := mc.Get(key)
	if err != nil {
		if err.Error() == "memcache: cache miss" {
			//fmt.Println("Couldn't find item in the cache")
			LogInDebugMode("Setting item: " + key + " in cache for: " + fmt.Sprint(SALE_CACHE_TIME_IN_SECS) + " seconds")
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
			LogInDebugMode("The prices haven't changed so we will not insert for id: ", itemId)
			return true
		}

		LogInDebugMode("The old price of: " + fmt.Sprint(s.Price) + " is different to: " + fmt.Sprint(price) + " busting the cache!")
		s.Price = price
		mc.Replace(&memcache.Item{Key: fmt.Sprint(key), Value: s.serialize(), Expiration: SALE_CACHE_TIME_IN_SECS})
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
