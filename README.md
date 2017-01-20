***Collection Service***
Responsible for receiving LogEvent messages from a LogClient and then persists them to the SQL database.   Auction Events are stored in the auction table and creates a record of the specific trader if they don't already exist.

Finally this service is responsible for talking to SQS to publish new LogClient events to all subscribers.

***Dev note***
All controllers are registered in `controllers.go` and all routes are registered in `routes.go`, every controller must conform to the Controller interface (which is currently empty) this interface is needed so we can do things like declare a map of Controllers.

**LICENSE**
Copyright 2017 - Alexander Sims
Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.