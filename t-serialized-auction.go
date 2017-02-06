package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type SerializedAuction struct {
	AuctionLine Auction
}

func (s *SerializedAuction) serialize() []byte {
	bytes, err := json.Marshal(s)
	if err != nil {
		fmt.Println("Error when marshaling")
	}

	return bytes
}

func (s *SerializedAuction) deserialize(bytes []byte) SerializedAuction {
	var sa SerializedAuction
	err := json.Unmarshal(bytes, &sa)
	if err != nil {
		fmt.Println("Error when unmarshaling")
	}

	return sa
}

func (s *SerializedAuction) toJSONString() []byte {
	itemMap := ""
	for _, item := range s.AuctionLine.Items {
		uri := TitleCase(strings.TrimSpace(item.Name), true)
		itemMap += `{ "name" : "` + TitleCase(strings.TrimSpace(item.Name), false) + `", "uri" : "` + uri + `" }, `
	}
	if len(itemMap) > 0 {
		itemMap = itemMap[0:len(itemMap)-2] // remove last ", "
	}

	outputString := `{ "Lines" : [{ "line" : "` + s.AuctionLine.Seller + " auctions, '" + s.AuctionLine.itemLine + `'", "items" : [` + itemMap + `] } ] }`

	return []byte(outputString)
}
