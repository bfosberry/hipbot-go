package main

// @botling nearby <query>
// Get 4 nearest places that respond to <query>
// return an HTML list of places with a map

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	GOOGLE_PLACES_ENDPOINT = "https://maps.googleapis.com/maps/api/place/nearbysearch/json"
	GOOGLE_MAPS_ENDPOINT   = "https://maps.googleapis.com/maps/api/staticmap"
)

var (
	googlePlacesParams = "?location=" + latLngPair + "&sensor=false&rankby=distance"
	googleMapsParams   = "?center=" + latLngPair + "&zoom=15&size=600x200&sensor=false"
	googleApiKey       = os.Getenv("GOOGLE_API_KEY")
)

var quotaGuardStaticURL *url.URL

func init() {
	envURL := os.Getenv("QUOTAGUARDSTATIC_URL")

	var err error
	quotaGuardStaticURL, err = url.ParseRequestURI(envURL)

	if err != nil {
		log.Println("DEBUG: env-url:", quotaGuardStaticURL)
		panic("Error parsing quotaguard url:" + err.Error())
	}
}

type Place struct {
	Icon      string      `json:"icon"`
	Name      string      `json:"name"`
	OpenHours OpenHours   `json:"opening_hours"`
	Rating    json.Number `json:"rating"`
	Address   string      `json:"vicinity"`
	Geometry  Geometry    `json:"geometry"`
}

type Geometry struct {
	PlaceLocation PlaceLocation `json:"location"`
}

type PlaceLocation struct {
	Lat json.Number `json:"lat"`
	Lng json.Number `json:"lng"`
}

type OpenHours struct {
	OpenNow bool `json:"open_now"`
}

type PlacesResponse struct {
	Places []Place `json:"results"`
}

// Get 4 nearest places pertaining to <query>
// HTML response includes a MAP(!) with markers of the 4 locations
func places(query string) string {
	additionalParams := "key=" + googleApiKey + "&keyword=" + url.QueryEscape(query)
	fullQueryUrl := GOOGLE_PLACES_ENDPOINT + googlePlacesParams + "&" + additionalParams

	// Sent GET request through proxy for static IP on heroku
	// -> for use with QuotaGuard
	transport := &http.Transport{Proxy: http.ProxyURL(quotaGuardStaticURL)}
	client := &http.Client{Transport: transport}

	res, err := client.Get(fullQueryUrl)
	if err != nil {
		log.Println("Error in HTTP GET:", err)
		return "error"
	}

	defer res.Body.Close()

	// Decode JSON response
	decoder := json.NewDecoder(res.Body)
	response := new(PlacesResponse)
	decoder.Decode(response)

	// Convert struct to a pretty HTML response with a map!
	return htmlPlaces(response.Places, query)
}

// return HTML, including a static Google MAP with (blue) markers of the 4 locations
func htmlPlaces(places []Place, query string) string {
	// Title
	html := "<strong>Results for Nearby " + strings.Title(query) + "</strong><br>"

	// Start unordered list
	html += "<ul>"

	// Initialize list of marker query params
	markers := ""

	// Only use the first 4 places
	for i := range places {
		if i > 3 {
			break
		}
		// Bullet point for each place, includes name, address, rating (or "N/A"), open-now
		html += "<li>" + places[i].Name + "<br>"
		html += places[i].Address + "<br>"
		html += "<em>Rating: " + stringRating(places[i].Rating) + "</em> | "
		html += openNowHtml(places[i].OpenHours.OpenNow) + "<br></li>"

		// Add marker for this place to the list
		markers += "&markers=color:blue|label:" + alphabet(i) + "|" + NewLatLngPair(places[i].Geometry.PlaceLocation)
	}

	// End list
	html += "</ul><br>"

	// Add static Google map
	html += "<img src='" + GOOGLE_MAPS_ENDPOINT + googleMapsParams + markers + "'>"

	return html
}

// Return a string representation of the Google+ rating
// If no rating, return "N/A"
func stringRating(rating json.Number) string {
	if len(rating) != 0 {
		return string(rating)
	} else {
		return "N/A"
	}
}

// Stringifies lat & lng and concatenates them together with a comma
func NewLatLngPair(location PlaceLocation) string {
	return (string(location.Lat) + "," + string(location.Lng))
}

// Return a string representation the boolean "open_now"
// If open - "Open Now", if closed - "Closed"
func openNowHtml(isOpen bool) string {
	if isOpen {
		return "<strong>Open Now</strong>"
	} else {
		return "<strong>Closed</strong>"
	}
}

// Maps an integer (0 - 6 ONLY) to an upper-case letter
func alphabet(i int) string {
	alphab := [7]string{"A", "B", "C", "D", "E", "F", "G"}
	return alphab[i]
}
