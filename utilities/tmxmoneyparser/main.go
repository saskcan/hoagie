package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// SearchResult represents the search result from tsxmoney.com
type SearchResult struct {
	LastUpdated int       `json:"last_updated"`
	Length      int       `json:"length"`
	Results     []*Symbol `json:"results"`
}

// Symbol represents a symbol
type Symbol struct {
	Symbol      string        `json:"symbol"`
	Name        string        `json:"name"`
	Instruments []*Instrument `json:"instruments"`
}

// Instrument represents an instrument
type Instrument struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

func main() {
	const OUTFILE = "output.csv"
	const BASEURL = "https://www.tsx.com/json/company-directory/search/tsx/"
	searchTerms := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z", "0-9"}

	f, err := os.Create(fmt.Sprintf("%s", OUTFILE))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	for _, term := range searchTerms {
		resp, err := http.Get(fmt.Sprintf("%s%%5E%s", BASEURL, term))
		if err != nil {
			log.Fatal(err)
		}

		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		var sr SearchResult
		err = json.Unmarshal(body, &sr)
		if err != nil {
			log.Fatal(err)
		}
		for _, sym := range sr.Results {
			for _, inst := range sym.Instruments {
				f.WriteString(fmt.Sprintf("%s,\"%s\"\n", inst.Symbol, inst.Name))
			}
		}
	}
}
