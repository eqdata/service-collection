package main

import (
	"testing"
	"time"
)

func TestToJSONStringItem1Qty1(t *testing.T) {
	auction := SerializedAuction{Auction{
		"bagel",
		time.Now(),
		[]Item{
			Item{
				"furry halberd",
				500.0,
				1,
				true,
				99,
			},
		},
		"blue",
		"Furry Halberd 500p", // ?
		"Bagel auctions 'WTS Furry Halberd 500p",
	}}

	actual := (&auction).toJSONString()

	expected := `{ "Lines" : [{ "line" : "bagel auctions, 'Furry Halberd 500p'", "items" : [{ "name" : "Furry Halberd", "uri" : "Furry_Halberd", "selling" : true, "price" : 500 }] } ] }`

	if string(actual) != expected {
		t.Errorf("\nExpected: %s\nActual:   %s", expected, string(actual))
	}
}

func TestToJSONStringItem1QtyN(t *testing.T) {
	auction := SerializedAuction{Auction{
		"bagel",
		time.Now(),
		[]Item{
			Item{
				"goblin ale",
				20.0,
				3,
				false,
				99,
			},
		},
		"blue",
		"Goblin Ale 3 x 20p", // ?
		"Bagel auctions 'Goblin Ale 3 x 20p",
	}}

	actual := (&auction).toJSONString()

	expected := `{ "Lines" : [{ "line" : "bagel auctions, 'Goblin Ale 3 x 20p'", "items" : [{ "name" : "Goblin Ale", "uri" : "Goblin_Ale", "selling" : false, "price" : 20, "qty" : 3 }] } ] }`

	if string(actual) != expected {
		t.Errorf("\nExpected: %s\nActual:   %s", expected, string(actual))
	}
}

func TestToJSONStringItemN(t *testing.T) {
	auction := SerializedAuction{Auction{
		"bagel",
		time.Now(),
		[]Item{
			Item{
				"goblin ale",
				20.0,
				3,
				true,
				99,
			},
			Item{
				"furry halberd",
				500.0,
				1,
				true,
				99,
			},
		},
		"blue",
		"Goblin Ale 3 x 20p", // ?
		"Bagel auctions 'Goblin Ale 3 x 20p",
	}}

	actual := (&auction).toJSONString()

	expected := `{ "Lines" : [{ "line" : "bagel auctions, 'Goblin Ale 3 x 20p'", "items" : [{ "name" : "Goblin Ale", "uri" : "Goblin_Ale", "selling" : true, "price" : 20, "qty" : 3 }, { "name" : "Furry Halberd", "uri" : "Furry_Halberd", "selling" : true, "price" : 500 }] } ] }`

	if string(actual) != expected {
		t.Errorf("\nExpected: %s\nActual:   %s", expected, string(actual))
	}
}
