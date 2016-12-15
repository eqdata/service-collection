package main

import (
	"fmt"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type Database struct {
	conn *sql.DB
}

func (d *Database) ConnectionString() string {
	return SQL_USER + ":" + SQL_PASS + "@tcp(" + SQL_HOST + ":" + SQL_PORT + ")/" + SQL_DB
}

func (d *Database) Open() bool {
	conn := d.ConnectionString();
	fmt.Println("Connecting to: ", conn)
	db, err := sql.Open("mysql", conn)
	if err != nil {
		fmt.Println(err.Error())
	}
	d.conn = db

	// Check that we can ping the DB box as the connection is lazy loaded when we fire the query
	err = d.conn.Ping()
	if err != nil {
		fmt.Println(err.Error())
	}

	return true
}

// Given a query string and a list of variadic parameters bindings this
// method will
func (d *Database) Query(query string, parameters ...interface{}) *sql.Rows {
	if d.conn == nil {
		fmt.Println("Spawning a new connection")
		d.Open()
	}

	fmt.Println("Interfaces: ", parameters)

	fmt.Println("Preparing query: " + query)
	stmt, err := d.conn.Prepare(query)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer stmt.Close()

	if len(parameters) > 0 {
		rows, err := stmt.Query(parameters...)
		if err != nil {
			fmt.Println("Error sending query: ", err)
			return nil
		}
		return rows
	} else {
		rows, err := stmt.Query()
		if err != nil {
			fmt.Println("Error sending query: ", err)
			return nil
		}
		return rows
	}
}

func (d *Database) Close() {
	if d.conn != nil {
		fmt.Println("Closing DB connection")
		err := d.conn.Close()
		if err == nil {
			fmt.Println("DB connection disposed successfully")
		} else {
			fmt.Println("Failed to close DB connection: ", err)
		}
	} else {
		fmt.Println("DB Connection was already closed")
	}
}