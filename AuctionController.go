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
	"sync"
	"bytes"
	"io/ioutil"
	"time"
)

type AuctionController struct {
	Controller
}

// Receive a list of auction lines from the Log client
func (c *AuctionController) store(w http.ResponseWriter, r *http.Request) {
	// Get api key and email
	var apiKey string = r.Header.Get("apiKey")
	var email string = r.Header.Get("email")
	var characterName string = r.Header.Get("characterName")

	// check for invalid credentials
	if len(strings.TrimSpace(apiKey)) < 14 || len(strings.TrimSpace(email)) == 5 || len(characterName) < 3 {
		if apiKey == "" { apiKey = "nil" }
		if email == "" { email = "nil" }

		fmt.Println("Invalid tokens")
		http.Error(w, "Please ensure you send a valid API Token and Email and Character Name. You provided email: " + email + ", API Key: " + apiKey + ", Character Name: " + characterName, 401)
		return
	}

	characterName = strings.ToLower(characterName)
	characterName = strings.Title(characterName)

	// Forward to the gatekeeper to see if this pair of items match
	req, err := http.NewRequest("GET", "http://" + GATEKEEPER_SERVICE_HOST + ":" + GATEKEEPER_SERVICE_PORT + "/auth", nil)
	if err != nil {
		http.Error(w, "Couldn't contact the gatekeeper service", 500)
		return
	}
	req.Header.Set("apiKey", apiKey)
	req.Header.Set("email", email)

	var client = &http.Client {
		Timeout: time.Second * 10,
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Could not reach the gatekeeper service", 500)
		return
	} else {
		defer resp.Body.Close()
		fmt.Println("we got here with resp: ", resp)
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

	go c.parse(&auctions, characterName)
}

func (c *AuctionController) isAuctionLine(line *string) bool {
	isValid, _ := regexp.MatchString("(\\[)([a-zA-Z0-9: ]+)(] ([A-Za-z]+) ((auction)|(auctions)))", *line)
	return isValid
}


// Gets a unique hash of the auction string and checks if it exists in memcached,
// if it does we parse the line, else we skip it
func (c *AuctionController) shouldParse(line *string) bool {

	// Create a 64bit hash key from this string
	hash := func(ln string) uint64 {
		h:= fnv.New64a()
		h.Write([]byte(ln))
		return h.Sum64()
	}(*line)

	// Check memcached to see if it exist
	mc := memcache.New(MC_HOST + ":" + MC_PORT)

	// Use an _ as we don't need to use the cache item returned
	_, err := mc.Get(fmt.Sprint(hash))
	if err != nil {
		if err.Error() == "memcache: cache miss" {
			fmt.Println("Setting hash: " + fmt.Sprint(hash) + " in cache for: " + fmt.Sprint(CACHE_TIME_IN_SECS) + " seconds")
			mc.Set(&memcache.Item{Key: fmt.Sprint(hash), Value: []byte(*line), Expiration: CACHE_TIME_IN_SECS})
			return true
		} else {
			fmt.Println("Error was: ", err.Error())
			return false
		}
	}

	// If we got here then we couldn't reach memcached, or there was a value
	// returned from memcached in which case we don't want to parse
	return false
}

// If we should parse this line, we send a list of items to the Wiki Service
// and then save unique auction data to the DB here (we do an initial save
// of the items name and display name here but don't process stats from the wiki)
func (c *AuctionController) parse(rawAuctions *RawAuctions, characterName string) {

	var outerWait sync.WaitGroup

	for _, line := range rawAuctions.Lines {

		outerWait.Add(1)
		go c.parseLine(line, characterName, &outerWait)

	}

	outerWait.Wait()
	fmt.Println("Processed all lines")
}

func (c *AuctionController) parseLine(line string, characterName string, wg *sync.WaitGroup) {
	if !c.isAuctionLine(&line) {
		wg.Done()
	} else {
		var auctions []Auction
		var auction Auction

		fmt.Println("Parsing line: ", line)

		// Split the auction string so we have date on the left and auctions on the right
		parts := strings.Split(line, "]")

		// Remove date stamp as this is localized to the users computer, we can't reliably
		// use this as the auction date time stamp because we can't reliably dictate if
		// the log-client is GMT, PST, EST etc.
		line = parts[1]

		parts[1] = strings.TrimSpace(parts[1])
		parts[1] = strings.Replace(parts[1], "You auction", (characterName + " auctions"), -1)

		auction.raw = parts[1]

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

		LogInDebugMode("Line is now: ", line)

		// trim any leading/trailing space so that we only explode string list on proper constraints
		items = strings.TrimSpace(items)

		fmt.Println("Items pre split: ", items);
		re := regexp.MustCompile(`(?i)wts|wtb|pst`)
		items = re.ReplaceAllString(items, "")
		re = regexp.MustCompile("((Spell: )?(([A-Z]{1,2}|(of|or|the|VP)?)[a-z]+[\\`']{0,1}[a-z]([-][a-z]+)?( {0,1})([I]{1,3})?)+([0-9]+(.[0-9]+)?[pkm]?)?|,-\\/&:)([\\d\\D]{1,3}(stacks|stack)){0,1}")
		itemList := re.FindAllString(items, -1)
		fmt.Println("Items after split: ", itemList)

		if !c.shouldParse(&line) {
			// If we can't parse then just append it to the relay server (could be the same  message)
			// dont do this yet, there is probably a better way of handling this!
			LogInDebugMode("Can't parse this line")
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
			LogInDebugMode("Seller is: ", seller)

			var wait sync.WaitGroup
			var itemsForWikiService []string
			for _, itemName := range itemList {
				wait.Add(1)
				// Make sure noone is trying to trade here
				orIndex := stringutil.CaseInsensitiveIndexOf(itemName, " or")
				if  orIndex > -1 {
					itemName = itemName[0:orIndex]
				}
				itemName = strings.TrimSpace(itemName)
				LogInDebugMode("Item is: " + itemName + ", length is: " + strconv.Itoa(len(itemName)))
				item := Item {
					Name: itemName,
				}
				go item.FetchData(func(raw Item) {
					auction.Items = append(auction.Items, raw)
					auction.Seller = seller

					fmt.Println("Parsed item is: ", raw)
					exists := stringutil.CaseInsensitiveSliceContainsString(itemsForWikiService, raw.Name)
					if !exists {
						itemsForWikiService = append(itemsForWikiService, raw.Name)
					}

					wait.Done()
				})
			}

			// Wait for all inner work to complete before we process next line
			wait.Wait()
			wg.Done()
			go c.saveAuctionData(auctions)
			go c.sendItemsToWikiService(itemsForWikiService)

			// Append to the output array and send it to the web front end (batching updates looks slow)
			auctions = append(auctions, auction)
			go c.publishToRelayService(auction)
		}
	}
}

//
func (c *AuctionController) sendItemsToWikiService(items []string) {
	if len(items) > 0 {
		fmt.Println("Sending: " + fmt.Sprint(len(items)) + " items to wiki service.")

		encodedItems, _ := json.Marshal(items)
		resp, err := http.Post("http://" + WIKI_SERVICE_HOST + ":" + WIKI_SERVICE_PORT + "/items", "application/json", bytes.NewBuffer(encodedItems))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Response from wiki service: ", resp)
		}
	}
}

// Publishes new auction data to Amazon SQS, this service is responsible
// for being the publisher in the pub/sub model, the Relay server
// is the subscriber which streams the data to the consumer via socket.io
func (c *AuctionController) saveAuctionData(auctions []Auction) {
	// Spawn all go save events:
	for _, auction := range auctions {
		a := auction
		go a.Save()
	}
}

func (c *AuctionController) publishToRelayService(auction Auction) {
	// Push to our Websocket server
	fmt.Println("Pushing: " + fmt.Sprint(len(auction.Items)) + " items in this auction to relay server.")

	// Serialize to JSON to pass to the Relay server
	sa := SerializedAuction{AuctionLine: auction}
	req, err := http.NewRequest("POST", "http://" + RELAY_SERVICE_HOST + ":" + RELAY_SERVICE_PORT + "/auctions", bytes.NewBuffer(sa.toJSONString()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	//fmt.Print("Sending req: ", req)

	var client = &http.Client {
		Timeout: time.Second * 10,
	}
	resp, err := client.Do(req)
	defer resp.Body.Close()
}
