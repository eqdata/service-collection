package main

import (
	"encoding/json"
	"fmt"
)

type SerializedAuctions struct {
	AuctionLines []Auction
}

func (s *SerializedAuctions) serialize() []byte {
	bytes, err := json.Marshal(s)
	if err != nil {
		fmt.Println("Error when marshaling")
	}

	return bytes
}

func (s *SerializedAuctions) deserialize(bytes []byte) SerializedAuctions {
	var sa SerializedAuctions
	err := json.Unmarshal(bytes, &sa)
	if err != nil {
		fmt.Println("Error when unmarshaling")
	}

	return sa
}

func (s *SerializedAuctions) toJSONString() []byte {
	var outputString string = `{ "Lines": [`

	for _, auction := range s.AuctionLines {
		outputString += `{ "line" : "` + auction.Seller + ` auctions: `
		for _, item := range auction.Items {
			if item.Name != "" {
				if item.Quantity > 1 {
					outputString += fmt.Sprint(item.Quantity) + "x "
				}
				uri := TitleCase(item.Name, true)
				outputString += "<a class='item' href='/" + uri + "'>" + item.Name + "</a> "

				if item.Price > 0.0 {
					var modifier string = "p"
					var newPrice float32 = item.Price
					if item.Price >= 1000000 {
						newPrice = newPrice / 1000000
						modifier = "m"
					} else if item.Price >= 1000 {
						newPrice = newPrice / 1000
						modifier = "k"
					}
					outputString += (fmt.Sprint(newPrice) + modifier)
				}
				outputString += " || "
			}
		}
		outputString = outputString[0:len(outputString)-3] // remove last ||
		outputString += `" },`
	}

	outputString = outputString[0:len(outputString)-1] // remove last ,
	outputString += "] }"

	fmt.Println(outputString)

	return []byte(outputString)
}
