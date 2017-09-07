package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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
	Name     string
	Price    float32
	Quantity int16
	selling  bool
	id       int64
}

// This method should be fairly self explanatory.  We simply use a regex to
// extract matches from the input string and then write the data back out
// to the last item on the input struct (this assumes that meta info is in
// the order of ITEM QUANTITY PRICE or ITEM PRICE QUANTITY etc.
// if the order is QUANTITY ITEM PRICE then the quantity will be assigned to
// the prior item (assuming auction.items > 1) and the price would be assigned
// to the correct item, if however auction.items == 0 then the extracted meta
// inf is lost, this could possibly be parsed correctly by storing a buffer
// of prices for previously parsed items when the legnth is 0, as we could then
// assume that the rest of the items would follow the same pattern in that string...(TODO?)
func (i *Item) ParsePriceAndQuantity(buffer *[]byte, auction *Auction) bool {
	//fmt.Println("Parsing: " + string(*buffer) + " for price data")
	price_regex := regexp.MustCompile(`(?im)^(x ?)?(\d*\.?\d*)(k|p|pp| ?x)?$`)
	price_string := strings.TrimSpace(string(*buffer))

	matches := price_regex.FindStringSubmatch(price_string)
	if len(matches) > 1 && len(strings.TrimSpace(matches[0])) > 0 && len(auction.Items) > 0 {
		matches = matches[1:]
		price, err := strconv.ParseFloat(strings.TrimSpace(matches[1]), 64)
		if err != nil {
			fmt.Println("error setting for string: "+strings.TrimSpace(matches[1])+", price: ", err)
			price = 0.0
		}
		var prelimiter string = strings.TrimSpace(strings.ToLower(matches[0]))
		var delimiter string = strings.ToLower(matches[2])
		var multiplier float64 = 1.0
		var isQuantity bool = false

		switch delimiter {
		case "x":
			isQuantity = true
			break
		case "p":
			multiplier = 1.0
			break
		case "k":
			multiplier = 1000.0
			break
		case "pp":
			multiplier = 1.0
			break
		case "m":
			multiplier = 1000000.0
			break
		default:
			multiplier = 1
			break
		}

		if prelimiter == "x" {
			isQuantity = true
		}

		// check if we need to set a new multiplier
		price_without_delim_regex := regexp.MustCompile(`(?im)^([0-9]{1,}\.[0-9]{1,})$`)
		matches = price_without_delim_regex.FindAllString(price_string, -1)
		if len(matches) > 0 {
			multiplier = 1000.0
		}

		// check if this was in-fact quantity data
		var item *Item = &auction.Items[len(auction.Items)-1]
		if isQuantity == true && price > 0.0 {
			//fmt.Println("setting quantity: ", fmt.Sprint(int16(price)))
			item.Quantity = int16(price)
		} else if price > 0.0 && float32(price*multiplier) > item.Price {
			item.Price = float32((price * multiplier) / float64(item.Quantity))
		}

		return true
	}

	return false
}
