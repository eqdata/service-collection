***Collection Service***
Responsible for receiving LogEvent messages from a LogClient and then persists them to the SQL database.   Auction Events are stored in the auction table and creates a record of the specific trader if they don't already exist.

Finally this service is responsible for talking to SQS to publish new LogClient events to all subscribers.

***Dev note***
All controllers are registered in `controllers.go` and all routes are registered in `routes.go`, every controller must conform to the Controller interface (which is currently empty) this interface is needed so we can do things like declare a map of Controllers.