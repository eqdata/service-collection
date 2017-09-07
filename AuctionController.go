package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alexmk92/stringutil"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/fvbock/trie"
	"hash/fnv"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type WalkResult struct {
	match      bool
	searchTerm string
	value      string
}

func (w *WalkResult) reset() {
	w.match = false
	w.searchTerm = ""
	w.value = ""
}

type AuctionController struct {
	Controller
	ItemTrie   *trie.Trie
	WalkResult WalkResult
}

// Receive a list of auction lines from the Log client
func (c *AuctionController) store(w http.ResponseWriter, r *http.Request) {
	// Get api key and email
	var apiKey string = r.Header.Get("apiKey")
	var email string = r.Header.Get("email")
	var characterName string = r.Header.Get("characterName")
	var serverType string = strings.TrimSpace(strings.ToUpper(r.Header.Get("serverName")))

	validServer := (serverType == "RED" || serverType == "BLUE")
	// check for invalid credentials
	if len(strings.TrimSpace(apiKey)) != 14 || len(strings.TrimSpace(email)) == 5 || len(characterName) < 3 || !validServer {
		if apiKey == "" {
			apiKey = "nil"
		}
		if email == "" {
			email = "nil"
		}

		http.Error(w, "Please ensure you send a valid API Token, Email, Character Name and specify RED or BLUE server. You provided email: "+email+", API Key: "+apiKey+", Character Name: "+characterName+", Server Name: "+serverType, 401)
		return
	}

	characterName = strings.ToLower(characterName)
	characterName = strings.Title(characterName)

	// Forward to the gatekeeper to see if this pair of items match
	req, err := http.NewRequest("GET", "http://"+GATEKEEPER_SERVICE_HOST+":"+GATEKEEPER_SERVICE_PORT+"/auth", nil)
	if err != nil {
		http.Error(w, "Couldn't contact the gatekeeper service", 500)
		return
	}
	req.Header.Set("apiKey", apiKey)
	req.Header.Set("email", email)

	var client = &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Could not reach the gatekeeper service", 500)
		return
	} else {
		defer resp.Body.Close()
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		bodyString := string(bodyBytes)

		if resp.StatusCode != 200 {
			fmt.Println("Response from wiki service: ", resp)
			http.Error(w, bodyString, resp.StatusCode)
			return
		}
	}

	// Do the auction processing
	var auctions RawAuctions
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	err = json.NewDecoder(r.Body).Decode(&auctions)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if len(auctions.Lines) == 0 {
		http.Error(w, "No lines were present in the auctions array", 400)
		return
	}

	go c.parse(&auctions, characterName, serverType)
}

func (c *AuctionController) isAuctionLine(line *string) bool {
	isValid, _ := regexp.MatchString("(\\[)([a-zA-Z0-9: ]+)(] ([A-Za-z]+) ((auction)|(auctions)))", *line)
	return isValid
}

// Gets a unique hash of the auction string and checks if it exists in memcached,
// if it does we parse the line, else we skip it
func (c *AuctionController) shouldParse(line *string, server string) bool {

	// Create a 64bit hash key from this string
	hash := func(ln string) uint64 {
		h := fnv.New64a()
		h.Write([]byte(ln))
		return h.Sum64()
	}(*line)

	// Check memcached to see if it exist
	mc := memcache.New(MC_HOST + ":" + MC_PORT)

	// Use an _ as we don't need to use the cache item returned
	key := (server + ":" + fmt.Sprint(hash))
	_, err := mc.Get(key)
	if err != nil {
		if err.Error() == "memcache: cache miss" {
			fmt.Println("Setting hash: " + fmt.Sprint(hash) + " in cache for: " + fmt.Sprint(CACHE_TIME_IN_SECS) + " seconds")
			mc.Set(&memcache.Item{Key: key, Value: []byte(*line), Expiration: CACHE_TIME_IN_SECS})
			return true
		} else {
			fmt.Println("Error was: ", err.Error())
			return false
		}
	}

	// If we got here then we couldn't reach memcached, or there was a value
	// returned from memcached in which case we don't want to parse
	fmt.Println("Key already exists: ", key)
	return false
}

// If we should parse this line, we send a list of items to the Wiki Service
// and then save unique auction data to the DB here (we do an initial save
// of the items name and display name here but don't process stats from the wiki)
func (c *AuctionController) parse(rawAuctions *RawAuctions, characterName, serverType string) {
	var auctions []Auction
	var outerWait sync.WaitGroup

	c.ItemTrie = trie.NewTrie()
	c.ItemTrie.Add("selling")
	c.ItemTrie.Add("buying")
	c.ItemTrie.Add("wtb")
	c.ItemTrie.Add("wts")
	c.ItemTrie.Add("ea")
	c.ItemTrie.Add("each")
	c.ItemTrie.Add("per")

	// Load all items into Trie structure
	itemQuery := "SELECT displayName, id FROM items ORDER BY displayName ASC"
	rows := DB.Query(itemQuery)
	if rows != nil {
		for rows.Next() {
			var itemName string
			var itemId int64

			rows.Scan(&itemName, &itemId)
			if itemId > 0 && itemName != "" {
				c.ItemTrie.Add(strings.ToLower(itemName))
			}
		}
	}

	for _, line := range rawAuctions.Lines {
		outerWait.Add(1)
		go c.parseLine(line, characterName, serverType, &outerWait, &auctions)
	}

	outerWait.Wait()
	fmt.Println("Processed all lines")

	if len(auctions) > 0 {
		c.saveAuctionData(auctions)
	}
}

// Extract
func (c *AuctionController) extractParserInformationFromLine(line string, auction *Auction) error {
	fmt.Println("Attempting to match: ", line)
	reg := regexp.MustCompile(`(?m)^\[(?P<Timestamp>[A-Za-z0-9: ]+)+] (?P<Seller>[A-Za-z]+) auction[s]?, '(?P<Items>.+)'$`)
	matches := reg.FindStringSubmatch(line)
	if len(matches) == 0 {
		return errors.New("No matches found for expression")
	}

	date := strings.TrimSpace(matches[1])
	layout := "Mon Jan 2 15:04:05 2006"
	t, err := time.Parse(layout, date)
	t.Format("02-01-2006 15:04:05")
	if err != nil {
		return errors.New("Invalid date stamp for this line, cannot parse!" + err.Error())
	}

	auction.raw = line
	//auction.Timestamp = matches[1]  only temp maybe
	auction.Seller = matches[2]
	auction.itemLine = matches[3]

	fmt.Println("Auction raw: " + auction.raw)
	fmt.Println("Auction seller: " + auction.Seller)
	fmt.Println("Auction line: " + auction.itemLine)

	return nil
}

// Pretty simple method, we query the trie in O(W * L) time to check
// if it the item can be appended, if not we check if it has a spell
// prefix and if it does we insert it to the trie, we insert with
// a base value of quant 1.0 then our pricing parser will fill in the
// rest
// TODO this could be optimised, we do some many checks here where we
// could probably optimised
func (c *AuctionController) appendIfInTrie(item *Item, out *[]Item) bool {
	if c.ItemTrie.Has(strings.TrimSpace(item.Name)) {
		item.Quantity = 1.0
		*out = append(*out, *item)
		return true
	} else if c.ItemTrie.Has("spell: " + strings.TrimSpace(item.Name)) {
		item.Name = "spell: " + item.Name
		item.Quantity = 1.0
		*out = append(*out, *item)
		return true
	} else if c.ItemTrie.Has("rune of " + strings.TrimSpace(item.Name)) {
		item.Name = "rune of " + item.Name
		item.Quantity = 1.0
		*out = append(*out, *item)
		return true
	} else if c.ItemTrie.Has("rune of the " + strings.TrimSpace(item.Name)) {
		item.Name = "rune of the " + item.Name
		item.Quantity = 1.0
		*out = append(*out, *item)
		return true
	} else if c.ItemTrie.Has("words of " + strings.TrimSpace(item.Name)) {
		item.Name = "words of " + item.Name
		item.Quantity = 1.0
		*out = append(*out, *item)
		return true
	} else if c.ItemTrie.Has("words of the " + strings.TrimSpace(item.Name)) {
		item.Name = "words of the " + item.Name
		item.Quantity = 1.0
		*out = append(*out, *item)
		return true
	}

	return false
}

// New parse line strategy, code is fairly self explanatory
func (c *AuctionController) parseLine(line, characterName, serverType string, wg *sync.WaitGroup, auctions *[]Auction) {
	fmt.Println(c.ItemTrie.Has("wurmslayer"))
	if !c.isAuctionLine(&line) {
		wg.Done()
	} else {
		auction := Auction{}
		item := Item{}

		auction.Server = serverType

		err := c.extractParserInformationFromLine(line, &auction)
		if err != nil {
			fmt.Println(err.Error())
			wg.Done()
			return
		} else {
			fmt.Println("Handling auction for seller: " + auction.Seller)
		}

		// check if we need to set the sellers name to the streaming clients name
		// this happens when the log detects a you auction: line.  We want
		// to supply the correct name for the auction DB otherwise sale data
		// is skewed tremendously!
		if strings.ToLower(auction.Seller) == "you" {
			auction.Seller = characterName
		}

		LogInDebugMode("Parsing line: ", line)

		cachedLine := auction.Seller + " auctions, '" + auction.itemLine + "'"
		if !c.shouldParse(&cachedLine, auction.Server) {
			// If we can't parse then just append it to the relay server (could be the same  message)
			// dont do this yet, there is probably a better way of handling this!
			fmt.Println("Can't parse this line: ", cachedLine)
			/*
				for _, itemName := range itemList {
					var item = Item{Name:itemName, Price:0.0, Quantity: 1, id:0}
					auction.Items = append(auction.Items, item)
				}
				auctions = append(auctions, auction)
				go c.publish(auctions, false)
			*/
			wg.Done()
		} else {
			//- Go one character at a time.
			//- Start in WTS mode (since some people just say /auc Ale)
			//- If we see WTB or "Buying" then switch to buying mode
			//- If we see WTS or "Selling" then switch to selling mode
			//- After consuming each character, check the items trie-test to see if anything
			//matches
			//- If the current characters aren't a prefix for anything, then throw away
			//the current characters and start processing again.
			//- If the current characters are a full match for an item, then register that
			//item.
			//- After finding an item, try to process the next characters as a price.
			//TODO: it's hard to reason about the matching strategy.  Make it simpler.
			//TODO: this doesn't support quantities like 'WTS Diamond x8 100pp each' or
			//'WTS Diamond (8) 8k'.  A quantity without an 'x' will be interpreted
			//as a price.
			//TODO: this does greedy matches, which means that it'll think things like
			//Yaulp IV are just plain old Yaulp.

			fmt.Println("Parsing line: ", line)

			buffer := []byte{}
			selling := true
			skippedChar := []byte{} // use an array so we can check the size

			// Don't deal with capitlization, remove it here (trie only checks lowercase)
			line = strings.ToLower(auction.itemLine)
			line = ReplaceMultiple(line, " ", ",", "&", "\\", "/")

			// NOTE: We use Go's `continue` kewyword to break execution flow instead of
			// chaining else-if's.  I personally find this more readable with the
			// comment blocks above each part of the parser!!
			var prevMatch string = ""
			for i, char := range strings.ToLower(line) {
				buffer = append(buffer, byte(char))

				// check for selling
				if stringutil.CaseInsenstiveContains(string(buffer), "wts", "selling") {
					selling = true
					buffer = []byte{}
					prevMatch = ""
					skippedChar = []byte{}
					continue
				}

				// check for buying
				if stringutil.CaseInsenstiveContains(string(buffer), "wtb", "buying", "trading") {
					selling = false
					buffer = []byte{}
					prevMatch = ""
					skippedChar = []byte{}
					continue
				}

				// check if we skipped a letter on the previous iteration and shift the items forward
				// this checks example: wurmslayerale it would fail at wurmslayera we set "a" as the
				// skipped character, extract wurmslayer and the begin to match ale using the "a" char
				// once we append to the buffer we reset the skipped char to avoid prepending on
				// subsequent calls
				if len(skippedChar) > 0 {
					buffer = append(skippedChar[0:1], buffer...)
					skippedChar = []byte{}
				}

				// Create a test string based on the current buffer but we stripped the prefix of
				// a or an from the front if we can't get a match on the initial buffer
				// This will allow us to still match things like A Shamanistic Shenannigan Doll
				nameWithoutPrefix := strings.ToLower(string(buffer))
				nameWithoutPrefix = strings.Replace(strings.TrimSpace(nameWithoutPrefix), "a ", "", -1)
				nameWithoutPrefix = strings.Replace(strings.TrimSpace(nameWithoutPrefix), "an ", "", -1)

				if !c.ItemTrie.HasPrefix(strings.TrimLeft(string(buffer), " ")) && c.ItemTrie.HasPrefix(strings.TrimLeft(nameWithoutPrefix, " ")) {
					buffer = []byte(nameWithoutPrefix)
				}

				if c.checkIfQuantityWasBasedOnEach(strings.TrimSpace(string(buffer))) && len(auction.Items) > 0 {
					auction.Items[len(auction.Items)-1].Price *= float32(auction.Items[len(auction.Items)-1].Quantity)
					buffer = []byte{}
					prevMatch = ""
					skippedChar = []byte{}
					continue
				}

				// check if the current string exists in the buffer, we trim any spaces
				// from the left but not the right as that can skew the results
				// if we find a match store the previous match, for the next iteration.
				// finally we check to see if we're at the last position in the line,
				// if we are then we reset the buffer and attempt to append to the trie
				// if our buffer contains a match
				//fmt.Println("checking if trie has: ", string(buffer))
				// TODO optimise the check for spell, rune, words etc. The method chaining
				// could probably be done with a single lookup method instead of chaining in the condition
				//
				// If we don't find any matches in the trie then we want to clear out the buffer
				// if the last character in the buffer is a space.  We do this because we still
				// want to try and parse price data which is not stored in the tree obviously.
				// If we don't clear the buffer then the parse can occasionally miss items
				// on its pass through
				if c.ItemTrie.HasPrefix(strings.TrimLeft(string(buffer), " ")) ||
					c.ItemTrie.HasPrefix("spell: "+strings.TrimLeft(string(buffer), " ")) ||
					c.ItemTrie.HasPrefix("words of "+strings.TrimLeft(string(buffer), " ")) ||
					c.ItemTrie.HasPrefix("words of the "+strings.TrimLeft(string(buffer), " ")) ||
					c.ItemTrie.HasPrefix("rune of "+strings.TrimLeft(string(buffer), " ")) ||
					c.ItemTrie.HasPrefix("rune of the "+strings.TrimLeft(string(buffer), " ")) {
					prevMatch = string(buffer)
					//fmt.Println("Has prefix: ", string(buffer))
					if i == len(line)-1 {
						buffer = []byte{}
						item.Name = prevMatch
						item.selling = selling
						c.appendIfInTrie(&item, &auction.Items)
						prevMatch = ""
						skippedChar = []byte{}
					}

					continue
				} else if string(buffer[len(buffer)-1]) == " " {
					buffer = []byte{}
				}
				// The trie did not have the prefix composed of the char buffer, we now evaluate
				// the "previousMatch" which is the buffer string n-1.  We can assume that
				// on this iteration the new character accessed caused the buffer to be
				// invalidated on the item trie, therefore we append this character
				// into the skippedChar byte and then clear the buffer.
				// On our next iteration we populate the buffer with this "skipped"
				// character in order to build the next item line...
				// this allows us to catch cases where items are budged
				// up against one another without separators such as
				// wurmslayerswiftwindale would allow us to extract:
				// wurmslayer swiftwind ale
				// NOTE: We don't reset the buffer in this method as we always want
				// to check for Pricing and Quantity data, we will only reset
				// the buffer if no match is found for meta information about
				// the current item!
				if prevMatch != "" {
					//fmt.Println("Prev was: ", prevMatch)

					// We don't want to put spaces back into the buffer, the whole purpose of
					// skippedChar is to catch cases where uses budge items together.
					// Therefore we will only append non space characters.
					if string(byte(char)) != " " {
						skippedChar = append(skippedChar, byte(char))
					}
					//fmt.Println("Buffer is: ", (string(buffer)))
					//fmt.Println("Skipped buffer is: ", string(skippedChar))

					item.Name = prevMatch
					item.selling = selling

					c.appendIfInTrie(&item, &auction.Items)
				}
				// This is the final part of the parser, the previous block will have added a
				// new item to the auction items array if it found a match in the trie, otherwise
				// the array will remain the same.
				// At this point we want to extract any meta information for this item.  We can
				// assume that the buffer now contains information like " x2 50p" which we want
				// to extract and assign to the item.   If however none of our price extraction
				// reg-exs find a match we set "prevMatch" back to null and we also empty our buffer
				// as we have now essentially exhausted our search for this item
				if !item.ParsePriceAndQuantity(&buffer, &auction) {
					prevMatch = ""
					buffer = []byte{}

					continue
				}
				// Just continue execution, nothing else to be caught here - this means that we have
				// successfully extracted meta information, woot!
				// NOTE: We reset the skippedChar buffer here as we found some meta information
				// on the price or quantity, therefore we dont need to append on the next
				// iteration of hte loop
				skippedChar = []byte{}
			}

			//fmt.Println("Buffer is: ", string(buffer))
			//fmt.Println("Is sell mode? ", selling)
			//fmt.Println("Items is: ", &auction.Items)
			//fmt.Println("Total items: ", fmt.Sprint(len(auction.Items)))

			itemsForWikiService := []string{}
			for _, item := range auction.Items {
				exists := stringutil.CaseInsensitiveSliceContainsString(itemsForWikiService, item.Name)
				if !exists {
					itemsForWikiService = append(itemsForWikiService, item.Name)
				}
			}
			//fmt.Println("Sending: " + fmt.Sprint(itemsForWikiService) + " to service")

			// Append to the output array and send it to the web front end (batching updates looks slow)
			*auctions = append(*auctions, auction)
			go c.publishToRelayService(auction)
			go c.sendItemsToWikiService(itemsForWikiService)

			wg.Done()
		}
	}
}

func (c *AuctionController) checkIfQuantityWasBasedOnEach(term string) bool {
	fmt.Println("Checking line: ", term)
	re := regexp.MustCompile(`[a-z]+`)
	match := re.FindStringSubmatch(term)
	if len(match) > 0 {
		term = match[0]
	}

	fmt.Println("Each is: ", term)
	return term == "ea" || term == "each" || term == "per"
}

// Publishes a list of items to the wiki service to fetch their stats
func (c *AuctionController) sendItemsToWikiService(items []string) {
	if len(items) > 0 {
		//fmt.Println("Sending: " + fmt.Sprint(len(items)) + " items to wiki service.")

		encodedItems, _ := json.Marshal(items)
		resp, err := http.Post("http://"+WIKI_SERVICE_HOST+":"+WIKI_SERVICE_PORT+"/items", "application/json", bytes.NewBuffer(encodedItems))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Response from wiki service: ", resp.StatusCode)
		}
	}
}

// Publishes new auction data to Amazon SQS, this service is responsible
// for being the publisher in the pub/sub model, the Relay server
// is the subscriber which streams the data to the consumer via socket.io
func (c *AuctionController) saveAuctionData(auctions []Auction) {
	// Spawn all go save events:
	//fmt.Println("Saving: " + fmt.Sprint(len(auctions)) + " auctions", auctions)
	// Removed timestamp temporarily, mayb eperm
	auctionQuery := "INSERT INTO auctions (player_id, item_id, price, quantity, server, raw_auction, for_sale) " +
		" VALUES "

	wg := sync.WaitGroup{}
	var auctionParams []interface{}
	for _, auction := range auctions {
		wg.Add(1)
		a := auction
		go a.ExtractQueryInformation(func(values string, parameters []interface{}) {
			if parameters != nil && values != "" {
				auctionParams = append(auctionParams, parameters...)
				auctionQuery += values
			}
			wg.Done()
		})
	}

	wg.Wait()

	auctionQuery = auctionQuery[0 : len(auctionQuery)-1]
	fmt.Println("Params are: ", auctionParams)
	fmt.Println("Query is: ", auctionQuery)
	if DB.conn != nil && len(auctionParams) > 0 {
		DB.Insert(auctionQuery, auctionParams...)
	}

	fmt.Println("Successfully saved: " + fmt.Sprint(len(auctionParams)/5) + " items for auction")
}

func (c *AuctionController) publishToRelayService(auction Auction) {
	// Push to our Websocket server
	fmt.Println("Pushing: " + fmt.Sprint(len(auction.Items)) + " items in this auction to relay server.")

	// Serialize to JSON to pass to the Relay server
	sa := SerializedAuction{AuctionLine: auction}
	req, err := http.NewRequest("POST", "http://"+RELAY_SERVICE_HOST+":"+RELAY_SERVICE_PORT+"/auctions/"+strings.ToLower(auction.Server), bytes.NewBuffer(sa.toJSONString()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	//fmt.Print("Sending req: ", req)

	var client = &http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Do(req)
	if err != nil {

	} else {
		defer resp.Body.Close()
	}
}
