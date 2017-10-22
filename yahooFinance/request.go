package yahooFinance

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/saskcan/common/types"
)

// dataRow represents a parsed row from the csv format provided by Yahoo Finance
type dataRow struct {
	Open      float32
	High      float32
	Low       float32
	Close     float32
	StartTime time.Time
}

// RetrieveData retrieves data for the given product, frequency and dates
func RetrieveData(sym types.Symbol, freq types.Frequency, rng types.DateRange) ([]*types.Candle, error) {
	if sym == "" {
		return nil, errors.New("sym missing")
	}

	reqURL, err := makeRequestURL(sym, freq, rng)
	if err != nil {
		return nil, err
	}

	client, err := getClient(reqURL)
	if err != nil {
		return nil, err
	}

	resp, err := client.Get(reqURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBites, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		errorBody := string(bodyBites)
		return nil, fmt.Errorf("Expected a status code of %v but got %v; Body is: %s", http.StatusOK, resp.StatusCode, errorBody)
	}

	csv := string(bodyBites)

	dataRows, err := parseCSV(csv)
	if err != nil {
		return nil, err
	}

	var candles []*types.Candle

	for _, row := range dataRows {
		candle, err := makeCandle(sym, freq, *row)
		if err != nil {
			return nil, err
		}
		candles = append(candles, candle)
	}

	return candles, nil
}

// makeRequestURL builds a URL for the symbol, frequency and date range provided
func makeRequestURL(sym types.Symbol, freq types.Frequency, rng types.DateRange) (*url.URL, error) {
	if sym == "" {
		return nil, errors.New("sym missing")
	}

	frqCode, err := getFrequencyCode(freq)
	if err != nil {
		return nil, err
	}

	// queryparam crumb must match cookie in client and was discovered through inspection of the web browser
	rawurl := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/download/%s?period1=%d&period2=%d&interval=%s&events=history&crumb=t5/yumui8w6", sym, rng.Start.Unix(), rng.End.Unix(), frqCode)

	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	return parsedURL, nil
}

// getFrequencyCode returns the correct value for the queryparam given a frequency
func getFrequencyCode(freq types.Frequency) (string, error) {
	switch freq {
	case types.MINUTE:
		return "1m", nil
	case types.HOUR:
		return "1h", nil
	case types.DAY:
		return "1d", nil
	case types.MONTH:
		return "1mo", nil
	case types.YEAR: // might not be supported
		return "year", nil
	default:
		return "", errors.New("Invalid freq")
	}
}

// getClient returns an http client with cookies and request URL already set
func getClient(reqURL *url.URL) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	// this cookie was discovered through inspection of the web browser
	cookie := http.Cookie{
		Name:     "B",
		Value:    "ebkqfodcl13l1&b=3&s=tf",
		Domain:   ".yahoo.com",
		Path:     "/",
		HttpOnly: false,
		Secure:   false,
	}

	jar.SetCookies(reqURL, []*http.Cookie{&cookie})

	var client = &http.Client{
		Timeout: time.Second * 10,
		Jar:     jar,
	}

	return client, nil
}

// parseCSV parses an input csv-formatted text into a list of dataRows
func parseCSV(input string) ([]*dataRow, error) {
	var dataRows []*dataRow

	if input == "" {
		return dataRows, nil
	}

	reader := csv.NewReader(strings.NewReader(input))

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	for idx, row := range rows {
		if idx > 0 {
			parsed, err := parseRow(row)
			if err != nil {
				return nil, err
			}

			dataRows = append(dataRows, parsed)
		}
	}

	return dataRows, nil
}

// parseRow parses a single row from a csv file into a dataRow
func parseRow(r []string) (*dataRow, error) {
	const numFields = 7 // date, open, high, low, close, adjClose, volume
	if len(r) != numFields {
		return nil, fmt.Errorf("Was expecting %d fields, but found %d", numFields, len(r))
	}

	startTime, err := parseDate(r[0])
	if err != nil {
		return nil, err
	}

	open, err := parsePrice(r[1])
	if err != nil {
		return nil, err
	}

	high, err := parsePrice(r[2])
	if err != nil {
		return nil, err
	}

	low, err := parsePrice(r[3])
	if err != nil {
		return nil, err
	}

	close, err := parsePrice(r[4])
	if err != nil {
		return nil, err
	}

	res := dataRow{
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		StartTime: *startTime,
	}

	return &res, nil
}

// parseDate parses a string into a date, using the expected format from Yahoo Finance
func parseDate(date string) (*time.Time, error) {
	const layout string = "2006-01-02"

	t, err := time.Parse(layout, date)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// parsePrice parses a string into a float32
func parsePrice(input string) (float32, error) {

	val, err := strconv.ParseFloat(input, 32)
	if err != nil {
		return 0, err
	}

	return float32(val), nil
}

// makeCandle returns a pointer to a Candle given a symbol, frequency and dataRow
func makeCandle(sym types.Symbol, freq types.Frequency, row dataRow) (*types.Candle, error) {
	if sym == "" {
		return nil, errors.New("Missing sym")
	}

	return &types.Candle{
		Open:      row.Open,
		High:      row.High,
		Low:       row.Low,
		Close:     row.Close,
		StartTime: row.StartTime,
		Symbol:    sym,
		Frequency: freq,
	}, nil
}
