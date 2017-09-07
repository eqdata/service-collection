package main

import (
	"encoding/json"
	"fmt"
	"strings"
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
		for _, item := range auction.Items {
			uri := TitleCase(item.Name, true)
			auction.raw = strings.Replace(auction.raw, item.Name, "<a class='item' href='/"+uri+"'>"+item.Name+"</a>", 1)
		}
		outputString += `{ "line" : "` + auction.raw + `" },`
	}

	outputString = outputString[0 : len(outputString)-1] // remove last ,
	outputString += "] }"

	LogInDebugMode(outputString)

	return []byte(outputString)
}
