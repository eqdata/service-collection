package main

import (
	"net/http"
	"fmt"
	"encoding/json"
	"strings"
	"regexp"
	"github.com/alexmk92/stringutil"
	"strconv"
	"hash/fnv"
	"github.com/bradfitz/gomemcache/memcache"
)

type AuctionController struct {
	Controller
}

// Stores auction data to the Amazon RDS storage once it has been parsed
func (c *AuctionController) store(w http.ResponseWriter, r *http.Request) {
	var auctions RawAuctions
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&auctions)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if len(auctions.Lines) == 0 {
		http.Error(w, "No lines were present in the auctions array", 400)
		return
	}

	go c.parse(&auctions)
}

func (c *AuctionController) shouldParse(line *string) bool {

	// Create a 64bit hash key from this string
	hash := func(ln string) uint64 {
		h:= fnv.New64a()
		h.Write([]byte(ln))
		return h.Sum64()
	}(*line)

	fmt.Println("The hash is: ", hash)

	// Check memcached to see if it exist
	mc := memcache.New(MC_HOST + ":" + MC_PORT)

	// Use an _ as we don't need to use the cache item returned
	_, err := mc.Get(fmt.Sprint(hash))
	if err != nil {
		if err.Error() == "memcache: cache miss" {
			fmt.Println("Setting in cache for: " + fmt.Sprint(CACHE_TIME_IN_SECS) + " seconds")
			mc.Set(&memcache.Item{Key: fmt.Sprint(hash), Value: []byte(*line), Expiration: CACHE_TIME_IN_SECS})
			return true
		} else {
			fmt.Println("Error was: ", err)
			return false
		}
	}

	// If we got here then we couldn't reach memcached, or there was a value
	// returned from memcached in which case we don't want to parse
	return false
}

// Publishes new auction data to Amazon SQS, this service is responsible
// for being the publisher in the pub/sub model, the Relay server
// is the subscriber which streams the data to the consumer via socket.io
func (c *AuctionController) publish() {
	fmt.Println("Pushing data to queue system")
}

//
func (c *AuctionController) parse(auctions *RawAuctions) {

	for _, line := range auctions.Lines {

		fmt.Println("Parsing line: ", line)

		// Split the auction string so we have date on the left and auctions on the right
		parts := strings.Split(line, "]")

		// Remove date stamp as this is localized to the users computer, we can't reliably
		// use this as the auction date time stamp because we can't reliably dictate if
		// the log-client is GMT, PST, EST etc.
		line = parts[1]

		parts[1] = strings.TrimSpace(parts[1])

		// Explode this array so we are left with the seller on the left and items on the right
		auctionParts := strings.Split(parts[1], "auctions,")

		seller := auctionParts[0]


		// Sale data is always encapsulated in single quotes, taking a substring removes these
		auctionParts[1] = strings.TrimSpace(auctionParts[1])[1:len(auctionParts[1])-2]

		line = auctionParts[0] + auctionParts[1]

		items := strings.TrimSpace(auctionParts[1])
		items = regexp.MustCompile(`(?i)wts`).ReplaceAllLiteralString(items, "")

		// Discard the WTB portion of the string
		wtbIndex := stringutil.CaseInsensitiveIndexOf(items, "WTB")
		if(wtbIndex > -1) {
			items = items[0:wtbIndex]
		}

		fmt.Println("Line is now: ", line)

		if !c.shouldParse(&line) {
			fmt.Println("Can't parse this line")
		} else {
			// trim any leading/trailing space so that we only explode string list on proper constraints
			items = strings.TrimSpace(items)
			itemList := strings.FieldsFunc(items, func(r rune) bool {
				switch r {
				case '|', ',', '-', ':', '/', '&':
					return true;
				}
				return false
			})

			fmt.Println("Seller is: ", seller)

			fetchChannel := make(chan bool)
			for _, itemName := range itemList {
				itemName = strings.TrimSpace(itemName)
				fmt.Println("Item is: " + itemName + ", length is: " + strconv.Itoa(len(itemName)))
				item := Item {
					name: itemName,
				}
				go item.FetchData(fetchChannel)
			}
		}
	}

	//c.publish()
}