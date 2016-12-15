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
 | @member name (string): Name of the item (url encoded)
 | @member displayName (string): Name of the item (browser friendly)
 | @member imageSrc (string): URL for the image stored on wiki
 | @member price (float32): The advertised price
 | @member statistics ([]Statistic): An array of all stats for this item
 |
 */

type Item struct {
	name string
	displayName string
	imageSrc string
	price float32
	statistics []Statistic
}

// Public method to fetch data for this item, in Go public method are
// capitalised by convention (doesn't actually enforce Public/Private methods in go)
// this method will call fetchDataFromWiki and fetchDataFromCache where appropriate
func (i *Item) FetchData(done chan <- bool) {
	fmt.Println("Fetching data for item: ", i.name)
	i.getPricingData()

	if(i.fetchDataFromSQL()) {
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
func (i *Item) fetchDataFromSQL() bool {
	var (
		name string
		displayName string
		statCode interface{}
		statValue interface{}
	)

	query := "SELECT name, displayName, code AS statCode, value AS statValue " +
		"FROM items " +
		"LEFT JOIN statistics " +
		"ON items.id = statistics.item_id " +
		"WHERE name = ? " +
		"OR displayName = ?"

	rows := DB.Query(query, i.name, i.name)
	if rows != nil {
		defer rows.Close()

		hasStat := false
		for rows.Next() {
			err := rows.Scan(&name, &displayName, &statCode, &statValue)
			if err != nil {
				fmt.Println("Scan error: ", err)
			}
			if statCode == nil || statValue == nil {
				fmt.Println("No stat exists for: ", displayName)
			} else {
				hasStat = true
			}
			fmt.Println("Row is: ", name, displayName, fmt.Sprint(statCode), fmt.Sprint(statValue))
		}
		return hasStat
	} else {
		fmt.Println("No rows found")
	}

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
		i.imageSrc = imageSrc

		// Extract the item information snippet
		openInfoParagraphIndex := stringutil.CaseInsensitiveIndexOf(body, "<p>") + 3 // +3 to ignore the <p> chars
		closeInfoParagraphIndex := stringutil.CaseInsensitiveIndexOf(body, "</p>")
		body = body[openInfoParagraphIndex:closeInfoParagraphIndex]

		upperParts := strings.Split(strings.TrimSpace(body), "<br />")
		fmt.Println(len(upperParts))

		for _, part := range upperParts {
			part = strings.TrimSpace(part)

			lowerParts := strings.Split(part, "  ")
			if(len(lowerParts) > 1) {
				for k :=0; k < len(lowerParts); k++ {
					i.assignStatistic(strings.TrimSpace(lowerParts[k]))
				}
			} else {
				i.assignStatistic(strings.TrimSpace(part))
			}
		}

		fmt.Println("Item is: ", i)

	} else {
		fmt.Println("No item data for this page")
	}
}

func (i *Item) assignStatistic(part string) {
	var stat Statistic

	fmt.Println("Assigning part: ", part)
	if stringutil.CaseInsenstiveContains(part, "lore item", "magic item", "temporary") {
		stat.code = "affinity"
		stat.effect = part
	} else if stringutil.CaseInsenstiveContains(part, "slot:", "class:", "race:", "size:", "skill:") {
		parts := strings.Split(part, ":")
		stat.code = strings.TrimSpace(parts[0])
		stat.effect = strings.TrimSpace(parts[1])
	} else if stringutil.CaseInsenstiveContains(part, "sv fire:", "sv cold:", "sv poison:", "sv magic:", "sv disease:", "dmg:", "ac:", "hp:", "dex:", "agi:", "sta:", "mana:", "cha:", "atk:", "wis:", "int:", "endr:", "wt:", "atk delay:") {
		parts := strings.Split(part, ":")

		isPositiveNumber := true
		if stringutil.CaseInsensitiveIndexOf(parts[1], "+") > -1 {
			parts[1] = strings.TrimSpace(strings.Replace(parts[1], "+", "", -1))
			isPositiveNumber = true
		} else if stringutil.CaseInsensitiveIndexOf(parts[1], "-") > -1 {
			parts[1] = strings.TrimSpace(strings.Replace(parts[1], "-", "", -1))
			isPositiveNumber = false
		}

		stat.code = strings.TrimSpace(parts[0])
		val, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 32)

		if err != nil {
			fmt.Println("Stat error: ", err)
		} else {
			stat.value = float32(val)
			if !isPositiveNumber {
				stat.value *= -1
			}
		}
	} else {
		fmt.Println("Unkown stat: ", part)
	}

	if stat.code != "" {
		i.statistics = append(i.statistics, stat)
	} else {
		fmt.Println("Nil stat code for: ", stat)
	}
}

// Saves the item to our SQL database
func (i *Item) Save() {

}