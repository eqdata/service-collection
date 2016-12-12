package main

/*
 |-------------------------------------------------------------------------
 | Type: Statistic
 |--------------------------------------------------------------------------
 |
 | Represents a statistic
 |
 | @member seller (string): The name of the stat (HP, Mana etc.)
 | @member value (int32): The value of the stat, this isn't a uint as we can
 | have negative stats, for example a fungi has -10 AGI or an AoN has -100HP
 |
 */

type Statistic struct {
	name string
	value int32
}
