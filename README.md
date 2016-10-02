gooru
=====
A clever mix between a booru image gallery and an imageboard forum, written in Go.

To get started, you'll want to do the following:
* `go get github.com/Circlepuller/gooru`
* Make a directory in `./public/` called `src/` (make sure it's writable by gooru, this is where uploads go!)
* Run `go build` as it will create a gooru executable.
* Edit config.json with your editor of choice. (If you don't want to use config.json, you can specify a configuration file with the `-config` flag when running gooru.)
* Run `./gooru init` to install the database and create an initial admin user.
* Since you should be all ready to go, run `./gooru run` and navigate to localhost:8080 (I plan on allowing different ports down the road.)

This software is still in development, and I don't recommend it for production use - there's more than likely a ton of bugs and whatnot.
