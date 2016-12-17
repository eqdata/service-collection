package main

import (
	"fmt"
	"strings"
	"github.com/alexmk92/stringutil"
	"regexp"
	"strconv"
	"sync"
)

/*
 |------------------------------------------------------------------
 | Type: Item
 |------------------------------------------------------------------
 |
 | Represents an item, when we fetch its data we first attempt to
 | hit our file cache, if the item doesn't exist there we fetch
 | it from the Wiki and then store it to our Mongo store
 |
 | @member name (string): Name of the item (url encoded)
 | @member displayName (string): Name of the item (browser friendly)
 | @member imageSrc (string): URL for the image stored on wiki
 | @member price (float32): The advertised price
 | @member statistics ([]Statistic): An array of all stats for this item
 |
 */

type Item struct {
	name string
	price float32
}

// Public method to fetch data for this item, in Go public method are
// capitalised by convention (doesn't actually enforce Public/Private methods in go)
// this method will call fetchDataFromWiki and fetchDataFromCache where appropriate
func (i *Item) FetchData(wg *sync.WaitGroup, out chan <- Item) {
	fmt.Println("Fetching data for item: ", i.name)
	i.getPricingData()

	if(i.fetchDataFromSQL()) {
		out <- Item{i.name, i.price}
		wg.Done()
	} else {
		i.Save()
		out <- Item{i.name, i.price}
		wg.Done()
	}
}

// Checks if the item has a price associated with its name set
// in the parser stage, if so amend the name to strip the price from
// it and set the price of the item on the struct
func (i *Item) getPricingData() {
	hasPricingData, _ := regexp.MatchString("([a-zA-Z ]+)([0-9]+[pk]?)", i.name)
	if hasPricingData {
		priceData := regexp.MustCompile("[0-9]+").FindAllString(i.name, -1)
		if len(priceData) > 0 {
			priceIndex := stringutil.CaseInsensitiveIndexOf(i.name, priceData[len(priceData)-1])
			if(priceIndex > -1) {
				var modifier float32 = 1.0
				// trim and get the last character so we can check the modifier
				compare := strings.ToLower(strings.TrimSpace(i.name)[len(i.name)-1:len(i.name)])
				if compare == "k" {
					modifier = 1000.0
				} else if compare == "m" {
					modifier = 1000000.0
				}

				i.name = TitleCase(strings.TrimSpace(i.name[0:priceIndex]), false)
				price, err := strconv.ParseFloat(priceData[len(priceData)-1], 32)
				if err == nil {
					i.price = float32(price) * modifier
				} else {
					panic(err)
				}
			}

			LogInDebugMode("Item is now: ", i)
		}
	}
}

// Check our cache first to see if the item exists - this will eventually return something
// other than a bool, it will return a parsed Item struct from a deserialised JSON object
// sent back from the mongo store
func (i *Item) fetchDataFromSQL() bool {
	var (
		name string
	)

	query := "SELECT name " +
		"FROM items " +
		"WHERE name = ? " +
		"OR displayName = ?"

	rows := DB.Query(query, i.name, i.name)
	if rows != nil {
		defer rows.Close()

		for rows.Next() {
			err := rows.Scan(&name)
			if err != nil {
				fmt.Println("Scan error: ", err)
			}
			return true
		}
	}

	LogInDebugMode("No record found in our SQL database for item: ", i.name)
	return false
}

// Very basic save functionality to save item to DB, main save will be done
// inside of the Wiki parser
func (i *Item) Save() {
	query := "INSERT IGNORE INTO items" +
		"(name, displayName)" +
		"VALUES (?, ?)"

	id, err := DB.Insert(query, TitleCase(i.name, false), TitleCase(i.name, true))
	if err != nil {
		fmt.Println(err.Error())
	} else if id == 0 {
		fmt.Println("Item already exists")
	} else if id > 0 {
		fmt.Println("Successfully created item: " + i.name + " with id: ", id)
	}
}
