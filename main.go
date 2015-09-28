package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dhconnelly/rtreego"
	"github.com/paulsmith/gogeos/geos"
	"github.com/voxelbrain/goptions"
)

// Holds the R-Tree used for searching data.
// The geographical objects stored in the R-Tree as described and searched
// through their bounding box. The goal of the R-Tree is only to quickly target
// which objects are near the point of interest.
var rt *rtreego.Rtree

// Holds the translation between ISO-3166-2 country code and country details
var countries_exp map[string]CountryDetails

// Interface to access the data embed in a rtreego.Spatial object
type SpatialData interface {
	rtreego.Spatial
	GetData() *GeoData
}

type GeoData struct {
	City          string         `json:"city"`
	CountryName   string         `json:"country"`
	CountryCode_2 string         `json:"country_iso3166-2"`
	CountryCode_3 string         `json:"country_iso3166-3"`
	Type          string         `json:"type"`
	Geom          *geos.Geometry `json:"-"`
}

// Represents the data which will be stored in the R-Tree
type GeoObj struct {
	bbox    *rtreego.Rect
	geoData *GeoData
}

// Function needed for storing data in the R-Tree
func (c *GeoObj) Bounds() *rtreego.Rect {
	return c.bbox
}

// Function needed for accessing the data from the SpatialData interface
func (c *GeoObj) GetData() *GeoData {
	return c.geoData
}

// Parses /rg/<lat>/<lng> or /rg/<lat>/<lng>/<precision> and returns a JSON describing the reverse geocoding
func reverseGeocodingHandler(w http.ResponseWriter, r *http.Request) {
	params := strings.Split(r.URL.Path[len("/rg/"):], "/")
	if len(params) != 2 && len(params) != 3 {
		http.Error(w, "Invalid format for lat/lon", http.StatusInternalServerError)
		return
	}

	lat, err := strconv.ParseFloat(params[0], 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	lng, err := strconv.ParseFloat(params[1], 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	precision := 1e-4
	if len(params) == 3 {
		precision, err = strconv.ParseFloat(params[2], 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	geodata, err := reverseGeocode(rt, lat, lng, precision)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if details, ok := countries_exp[geodata.CountryCode_2]; ok {
		geodata.CountryName = details.Name
		geodata.CountryCode_3 = details.ISO3166_3
	}
	b, err := json.Marshal(geodata)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(b)
}

// ---------------------------------------------------------------------------

func main() {
	daddr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:8080")
	if err != nil {
		log.Fatal(err)
	}

	options := struct {
		CityFiles    []string      `goptions:"-d, description='Data files to load'"`
		CountryNames string        `goptions:"-c, description='CSV file holding country names'"`
		Help         goptions.Help `goptions:"-h, --help, description='Show this help'"`
		ListenAddr   *net.TCPAddr  `goptions:"-l, --listen, description='Listen address for HTTP server'"`
	}{
		CountryNames: "data/countries_en.csv",
		ListenAddr:   daddr,
	}
	goptions.ParseAndFail(&options)

	countries_exp, err = load_country_names(options.CountryNames)
	if err != nil {
		log.Fatal(err)
	}

	rt = rtreego.NewTree(2, 25, 50)
	total_loaded_cities := 0
	start_t := time.Now()
	for _, fname := range options.CityFiles {
		loaded, err := load_gisgraphy_cities_csv(rt, fname)
		if err != nil {
			log.Fatal(err)
		}
		total_loaded_cities += loaded
	}
	log.Println("Loaded", total_loaded_cities, "cities in", time.Now().Sub(start_t))

	log.Println("Starting HTTP server on", options.ListenAddr)
	http.HandleFunc("/rg/", reverseGeocodingHandler)
	err = http.ListenAndServe(options.ListenAddr.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
}
