package main

import (
	"fmt"
	"strings"
	"net/http"
	"io/ioutil"
	"github.com/alexmk92/stringutil"
	"regexp"
	"strconv"
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
 | @member name (string): Name of the item
 | @member price (float32): The advertised price
 | @member types ([]string): Type of item, is it MAGIC ITEM, LORE etc.
 | @member slot ([]string): The slots this item can go in i.e. Primary & Secondary
 | @member skill (string): Is this 1h/2h slash etc. if empty its armor
 | @member delay (uint8): The delay of the weapon, if 0 its armor
 | @member statistics ([]Statistic): An array of all stats for this item
 | @member classes ([]string): An array of all classes than can use this item
 | @member races ([]string): An array of all races that can use this item
 | @member weight (uint8): How much this item weighs
 | @member size (string): Is this item SMALL, MEDIUM, LARGE etc.
 |
 */

type Item struct {
	name string
	price float32
	types []string
	slot []string
	skill string
	delay uint8
	statistics []Statistic
	classes []string
	races []string
	weight uint8
	size string
}

// Public method to fetch data for this item, in Go public method are
// capitalised by convention (doesn't actually enforce Public/Private methods in go)
// this method will call fetchDataFromWiki and fetchDataFromCache where appropriate
func (i *Item) FetchData(done chan <- bool) {
	fmt.Println("Fetching data for item: ", i.name)
	i.getPricingData()
	return

	if(i.fetchDataFromCache()) {
		fmt.Println("It exists in cache already")
		done <- true
	} else {
		i.fetchDataFromWiki()
		done <- true
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
			fmt.Println(i.name)
			priceIndex := stringutil.CaseInsensitiveIndexOf(i.name, priceData[len(priceData)-1])
			if(priceIndex > -1) {
				var modifier float32 = 1.0
				// trim and get the last character so we can check the modifier
				compare := strings.ToLower(strings.TrimSpace(i.name)[len(i.name)-1:len(i.name)])
				if compare == "k" {
					modifier = 1000.0
				}

				i.name = strings.TrimSpace(i.name[0:priceIndex])
				price, err := strconv.ParseFloat(priceData[len(priceData)-1], 32)
				if err == nil {
					i.price = float32(price) * modifier
				} else {
					panic(err)
				}
			}

			fmt.Println("Item is now: ", i)
		}
	}
}

// Data didn't exist on our server, so we hit the wiki here
func (i *Item) fetchDataFromWiki() {

	uriParts := strings.Split(i.name, " ")
	fmt.Println("URI PARTS ARE: ", uriParts)

	uriString := ""
	for _, part := range uriParts {
		compare := strings.ToLower(part)
		if(compare == "the" || compare == "of" || compare == "or" || compare == "and" || compare == "a" || compare == "an" || compare == "on" || compare == "to") {
			part = strings.ToLower(part)
		} else {
			part = strings.Title(part)
		}
		uriString += part + "_"
	}
	uriString = uriString[0:len(uriString)-1]

	fmt.Println("Requesting data from: ", WIKI_BASE_URL + "/" + uriString)

	resp, err := http.Get(WIKI_BASE_URL + "/" + uriString)
	if(err != nil) {
		fmt.Println("ERROR GETTING DATA FROM WIKI: ", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if(err != nil) {
		fmt.Println("ERROR EXTRACTING BODY FROM RESPONSE: ", err)
	}
	i.extractItemDataFromHttpResponse(string(body))
}

// Check our cache first to see if the item exists - this will eventually return something
// other than a bool, it will return a parsed Item struct from a deserialised JSON object
// sent back from the mongo store
func (i *Item) fetchDataFromCache() bool {
	return false
}

// Extracts data from body
func (i *Item) extractItemDataFromHttpResponse(body string) {
	itemDataIndex := stringutil.CaseInsensitiveIndexOf(body, "itemData")
	endOfItemDataIndex := stringutil.CaseInsensitiveIndexOf(body, "itembotbg")

	if(itemDataIndex > -1 && endOfItemDataIndex > -1) {

		body = body[itemDataIndex:endOfItemDataIndex]

		// Extract the item image - this assumes that the format is consistent (tested with 30 items thus far)
		imageSrc := body[stringutil.CaseInsensitiveIndexOf(body, "/images"):stringutil.CaseInsensitiveIndexOf(body, "width")-2]
		fmt.Println("IMAGE SRC IS: ", imageSrc)

		// Extract the item information snippet
		openInfoParagraphIndex := stringutil.CaseInsensitiveIndexOf(body, "<p>") + 3 // +3 to ignore the <p> chars
		closeInfoParagraphIndex := stringutil.CaseInsensitiveIndexOf(body, "</p>")
		body = body[openInfoParagraphIndex:closeInfoParagraphIndex]

		upperParts := strings.Split(strings.TrimSpace(body), "<br />")
		fmt.Println(len(upperParts))
		fmt.Println(upperParts)

		fmt.Println("Printing lower parts")
		for _, part := range upperParts {
			lowerParts := strings.Split(part, "  ")
			if(len(lowerParts) > 1) {
				for i :=0; i < len(lowerParts); i++ {
					fmt.Println("Part is: ", strings.TrimSpace(lowerParts[i]))
				}
			} else {
				fmt.Println("Part single is: ", strings.TrimSpace(part))
			}
		}

	} else {
		fmt.Println("No item data for this page")
	}
}