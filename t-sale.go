package main

import (
	"encoding/json"
	"fmt"
)

// Used to represent a single unit of a sale
type Sale struct {
	Seller   string
	ItemId   int64
	Price    float32
	Quantity int32
}

func (s *Sale) serialize() []byte {
	bytes, err := json.Marshal(s)
	if err != nil {
		fmt.Println("Error marshaling the sale: ", err)
	}

	LogInDebugMode("Marshaled the sale data: ", bytes)
	return bytes
}

func (s *Sale) deserialize(bytes []byte) Sale {
	var sale Sale

	err := json.Unmarshal(bytes, &sale)
	if err != nil {
		fmt.Println("Error when unmarshaling sale data: ", err)
	}

	LogInDebugMode("Unmarshaled sale data: ", sale)
	return sale
}
