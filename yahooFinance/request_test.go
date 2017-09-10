package yahooFinance

import (
	"testing"
	"time"

	"github.com/saskcan/finance/common"
)

func TestRetrieveData(t *testing.T) {
	var sym common.Symbol = "MSFT"
	var freq common.Frequency = common.MONTH
	rng := common.DateRange{Start: time.Now(), End: time.Now()}

	_, err := RetrieveData(sym, freq, rng)
	if err != nil {
		t.Error("Error retrieving data")
	}

	var missing common.Symbol = ""
	_, err = RetrieveData(missing, freq, rng)
	if err == nil {
		t.Error("Should not be able to retrieve data with missing symbol")
	}
}

func TestMakeRequestURL(t *testing.T) {
	var sym common.Symbol = "MSFT"
	var freq common.Frequency = common.MONTH
	rng := common.DateRange{Start: time.Date(2001, 1, 1, 5, 0, 0, 0, time.UTC), End: time.Date(2010, 1, 1, 5, 0, 0, 0, time.UTC)}

	url, err := makeRequestURL(sym, freq, rng)

	if err != nil {
		t.Error("Not able to make the request URL")
	}

	expected := "https://query1.finance.yahoo.com/v7/finance/download/MSFT?period1=978325200&period2=1262322000&interval=1mo&events=history&crumb=t5/yumui8w6"

	if url.String() != expected {
		t.Errorf("Got %s but expected %s", url, expected)
	}
}

func TestGetFrequencyCode(t *testing.T) {
	code, err := getFrequencyCode(common.MONTH)
	if err != nil {
		t.Error("Unable to get frequency code for a month")
	}

	if code != "1mo" {
		t.Errorf("Expected '1mo' but got %s", code)
	}
}
