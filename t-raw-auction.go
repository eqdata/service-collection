package main

/*
 |-------------------------------------------------------------------------
 | Type: RawAuctions
 |--------------------------------------------------------------------------
 |
 | Represents a raw auction item sent from the log-client
 |
 | @member zone (string): The name of the zone that the user is streaming from
 | @member lines ([]string): An array of strings containing the auction data
 |
 */

type RawAuctions struct {
	Zone string
	Lines []string
}