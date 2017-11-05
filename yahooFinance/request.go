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

	"github.com/saskcan/hoagie/types"
)

// dataRow represents a parsed row from the csv format provided by Yahoo Finance
type dataRow struct {
	Open      float32
	High      float32
	Low       float32
	Close     float32
	StartTime time.Time
	Volume    uint
}

// RetrieveData retrieves data for the given product, frequency and dates
func RetrieveData(sym string, freq string, start time.Time, end time.Time) ([]*types.Candle, error) {
	if sym == "" {
		return nil, errors.New("sym missing")
	}

	reqURL, err := makeRequestURL(sym, freq, start, end)
	if err != nil {
		return nil, err
	}

	fmt.Printf("URL is: %v\n", reqURL.String())
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

	for i, row := range dataRows {
		// last two candles are always incomplete
		if i == (len(dataRows) - 2) {
			fmt.Printf("broke for index: %d\n", i)
			break
		}
		candle, err := makeCandle(sym, freq, *row)
		if err != nil {
			return nil, err
		}
		candles = append(candles, candle)
	}

	return candles, nil
}

// makeRequestURL builds a URL for the symbol, frequency and date range provided
func makeRequestURL(sym string, freq string, start time.Time, end time.Time) (*url.URL, error) {
	if sym == "" {
		return nil, errors.New("sym missing")
	}

	frqCode, err := getFrequencyCode(freq)
	if err != nil {
		return nil, err
	}

	// queryparam crumb must match cookie in client and was discovered through inspection of the web browser
	rawurl := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/download/%s.TO?period1=%d&period2=%d&interval=%s&events=history&crumb=t5/yumui8w6", sym, start.Unix(), end.Unix(), frqCode)

	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	return parsedURL, nil
}

// getFrequencyCode returns the correct value for the queryparam given a frequency
func getFrequencyCode(freq string) (string, error) {
	switch freq {
	case "minute":
		return "1m", nil
	case "hour":
		return "1h", nil
	case "day":
		return "1d", nil
	case "month":
		return "1mo", nil
	case "year": // might not be supported
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

	vol, err := parseVolume(r[6])

	res := dataRow{
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		StartTime: *startTime,
		Volume:    vol,
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

func parseVolume(input string) (uint, error) {
	val, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(val), nil
}

// makeCandle returns a pointer to a Candle given a symbol, frequency and dataRow
func makeCandle(sym string, freq string, row dataRow) (*types.Candle, error) {
	if sym == "" {
		return nil, errors.New("Missing sym")
	}

	return &types.Candle{
		Open:      row.Open,
		High:      row.High,
		Low:       row.Low,
		Close:     row.Close,
		Date:      row.StartTime,
		Symbol:    sym,
		Frequency: freq,
		Volume:    row.Volume,
		//Exchange:  "tsx",
	}, nil
}
