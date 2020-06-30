package main

import(
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	max := 400000
	scanner.Buffer(make([]byte, max), max)

	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		txt := scanner.Text()

		n := 0
		for _,f := range ExtractFlightMessages(txt) {
			ProcessFlightMessage(&f)
			n++
		}
	}
}

func ProcessFlightMessage(f *Flight) {
	if f.Source == "TH" {
		// TH == Track Information; every 12s.
		fmt.Printf("%v\n", f)
	}
}

func ExtractFlightMessages(str string) []Flight {
	sMsg := Ns5MessageCollectionSingleMessage{}
	mMsg := Ns5MessageCollectionMultiMessage{}

	// We need to try both single and multi, as they are incompatible in what type they
	// assign to the `message` JSON field. Do multi first, as they're most common.
	if err := json.Unmarshal([]byte(str), &mMsg); err == nil {
		flights := []Flight{}
		for _,m := range mMsg.Ns5MessageCollection.Message {
			flights = append(flights, m.Flight)
		}
		return flights

	} else if err2 := json.Unmarshal([]byte(str), &sMsg); err2 == nil {
		return []Flight{sMsg.Ns5MessageCollection.Message.Flight}

	} else {
		// If you start to care about the per-field error handling,
		// https://www.alexedwards.net/blog/how-to-properly-parse-a-json-request-body
		// https://golang.org/src/encoding/json/decode.go
		// fmt.Errorf("ParseFlightMessage: not multi or single: <%v>, <%v>", err, err2)
		return []Flight{}
	}
}
