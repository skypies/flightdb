# flightdb - a database for flight tracks

Prerequisites:
* the [Go programming language](https://golang.org/dl/)
* the [Go appengine SDK](https://cloud.google.com/appengine/docs/go/), and add it to your `$PATH`
* define your Go workspace: `export GOPATH=~/go`

Download and run things locally
* `go get github.com/skypies/flightdb/app/frontend` (pulls down all dependencies)
* `goapp serve $GOPATH/github.com/skypies/flightdb/app/frontend` (build & run locally)
* Look at <http://localhost:8080/> (appengine admin panel is <http://localhost:8000/>)

To deploy everything into a Google Cloud project:

    $ goapp deploy              app/frontend
    $ goapp deploy              app/backend

    $ appcfg.py update_cron     app/backend
    $ appcfg.py update_indexes  app/backend
    $ appcfg.py update_queues   app/backend
    $ appcfg.py update_dispatch app/backend
