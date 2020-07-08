package backends

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"

	"github.com/schachmat/wego/iface"
)

type nwsConfig struct {
}

type nwsResponse struct {
    Temperature nwsResponseField
    Dewpoint    nwsResponseField
    RelativeHumidity nwsResponseField
    ApparentTemperature nwsResponseField
    WindDirection nwsResponseField
    WindSpeed nwsResponseField
    WindGust nwsResponseField
    ProbabilityOfPrecipitation nwsResponseField
    Visibility nwsResponseField
    Weather []nwsResponseWeatherField
}

type nwsResponsePoint struct {
    ValidTime string
    Value float32
}

type nwsResponseField struct {
    UoM string
    Values []nwsResponsePoint
}

type nwsResponseWeatherField struct {
    Coverage string
    Weather string
    Intensity string
}

type nwsGrid struct {
	wfo string
	x   int
	y   int
}

const (
	// https://www.weather.gov/documentation/services-web-api
	nwsPointURI    = "https://api.weather.gov/points/%.4f,%.4f"
	nwsCurrentURI  = "https://api.weather.gov/gridpoints/%s/%d/%d"
	nwsForecastURI = "https://api.weather.gov/gridpoints/%s/%d/%d/forecast"
	nwsUserAgent   = "wego-nws"
)

// converts latitude and longitude to grid coordinates that the api
// get get information about
func (c *nwsConfig) fetchGrid(lat float32, lon float32) (*nwsGrid, error) {

	url := fmt.Sprintf(nwsPointURI, lat, lon)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Could not get grid response %v", err)
	} else if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Bad response code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Unable to read response, %v", err)
	}

	var parsed map[string]nwsGrid
	if err = json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal response (%s): %v\nThe json body is: %s", url, err, string(body))
	}

	gridData := parsed["properties"]

	return &gridData, nil
}

func (c *nwsConfig) parseForecast(responsePoints map[string][]nwsResponsePoint) ([]iface.Day, error) {
    return nil, nil
}

func (c *nwsConfig) fetchForecast(grid *nwsGrid) ([]nwsResponse, error) {
	url := fmt.Sprintf(nwsForecastURI, grid.wfo, grid.x, grid.y)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Could not get grid response %v", err)
	} else if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Bad response code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Unable to read response, %v", err)
	}

	var parsed map[string]interface{}
	if err = json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("Unable to unmarshal response (%s): %v\nThe json body is: %s", url, err, string(body))
	}

	forecastData := parsed["properties"].([]nwsResponse)

	return forecastData, nil
}

func (c *nwsConfig) Setup() {
}

func (c *nwsConfig) Fetch(location string, numdays int) iface.Data {
	var ret iface.Data

	if matched, err := regexp.MatchString(`^-?[0-9]*(\.[0-9]+)?,-?[0-9]*(\.[0-9]+)?$`, location); !matched || err != nil {
		log.Fatalf("Error: The weather.gov backend only supports latitude,longitude pairs as location.\nInstead of `%s` try `40.748,-73.985` for example to get a forecast for New York", location)
	}

	var lat, lon float32
	count, err := fmt.Sscanf(location, "%f,%f", &lat, &lon)
	if count != 2 || err != nil {
		return iface.Data{}
	}

	grid, err := c.fetchGrid(lat, lon)
	if err != nil {
		log.Fatalf("Error: failed to retrieve weather.gov grid data for %s", location)
	}

    rawForecast, err := c.fetchForecast(grid)
	if err != nil {
		log.Fatalf("Error: could not retrieve forecast")
	}

    print(rawForecast)

    ret.GeoLoc = &iface.LatLon{Latitude: lat, Longitude: lon}

	return ret
}

func init() {
	iface.AllBackends["weather.gov"] = &nwsConfig{}
}
