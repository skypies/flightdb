package main

import(
	"fmt"
	_ "github.com/skypies/flightdb/ui"
	_ "github.com/skypies/flightdb/app/backend"
	_ "github.com/skypies/flightdb/app/frontend"
)

func main() {
	fmt.Printf("Hello!\n")
}
