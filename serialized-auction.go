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
	var outputString string = `{ "Lines": [`

	for _, item := range s.AuctionLine.Items {
		uri := TitleCase(item.Name, true)
		s.AuctionLine.raw = strings.Replace(s.AuctionLine.raw, item.Name, "<a class='item' href='/" + uri + "'>" + item.Name + "</a>", 1)
	}
	outputString += `{ "line" : "` + s.AuctionLine.raw + `" }`
	outputString += "] }"

	fmt.Println(outputString)

	return []byte(outputString)
}
