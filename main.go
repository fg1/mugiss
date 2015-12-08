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
var rt_countries *rtreego.Rtree

// Holds the translation between ISO-3166-2 country code and country details.
// This map can also be used to translate country names.
var countries_exp map[string]CountryDetails

// Interface to access the data embed in a rtreego.Spatial object
type SpatialData interface {
	rtreego.Spatial
	GetData() *GeoData
}

type GeoData struct {
	Id            int64          `json:"id"`
	City          string         `json:"city,omitempty"`
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

func suggestDownload(country_code2 string) {
	if details, ok := countries_exp[country_code2]; ok {
		log.Println("No city data found for", details.Name)
		log.Println("Download here: http://download.gisgraphy.com/openstreetmap/csv/cities/" + country_code2 + ".tar.bz2")
	}
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
	if err == ErrNoMatchFound {
		// Fallback to countries
		geodata, err = reverseGeocode(rt_countries, lat, lng, precision)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		suggestDownload(geodata.CountryCode_2)
	} else if err != nil {
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

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// Parses /gj/<south>/<west>/<north>/<east>.json and returns a GeoJSON containing the object currently loaded
func serveGeoJson(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path[len("/gj/"):]
	if strings.HasSuffix(url, ".json") {
		url = url[:len(url)-len(".json")]
	}

	params := strings.Split(url, "/")
	if len(params) != 4 {
		http.Error(w, "Invalid format for bbox", http.StatusInternalServerError)
		return
	}

	// String to float convertion
	paramsf := make([]float64, 4)
	for i, v := range params {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		paramsf[i] = f
	}

	bb, err := rtreego.NewRect(rtreego.Point{paramsf[1], paramsf[0]},
		[]float64{paramsf[3] - paramsf[1], paramsf[2] - paramsf[0]})
	if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
	}

	// results := rt_countries.SearchIntersect(bb)
	results := rt.SearchIntersect(bb)

	gjc := gjFeatureCollection{
		Type:     "FeatureCollection",
		Features: make([]gjFeature, len(results)),
	}

	for i, res := range results {
		obj, _ := res.(SpatialData)
		geod := obj.GetData()
		gjf, err := ToGjFeature(geod.Geom)
		if err != nil {
			continue
		}
		gjc.Features[i] = gjf
		gjc.Features[i].Properties = *geod
	}

	b, err := json.Marshal(gjc)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(b)
}

// ---------------------------------------------------------------------------

func main() {
	daddr, err := net.ResolveTCPAddr("tcp", "0.0.0.0:8080")
	if err != nil {
		log.Fatal(err)
	}

	options := struct {
		CityFiles     []string      `goptions:"-d, description='Data files to load'"`
		CountryNames  string        `goptions:"-c, description='CSV file holding country names'"`
		CountryShapes string        `goptions:"-c, description='CSV file holding country shapes'"`
		Help          goptions.Help `goptions:"-h, --help, description='Show this help'"`
		ListenAddr    *net.TCPAddr  `goptions:"-l, --listen, description='Listen address for HTTP server'"`
	}{
		CountryNames:  "data/countries_en.csv",
		CountryShapes: "data/countries.csv.bz2",
		ListenAddr:    daddr,
	}
	goptions.ParseAndFail(&options)

	rt_countries = rtreego.NewTree(2, 10, 20)
	_, err = load_freegeodb_countries_csv(rt_countries, options.CountryShapes)
	if err != nil {
		log.Fatal(err)
	}

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
	if total_loaded_cities > 0 {
		log.Println("Loaded", total_loaded_cities, "cities in", time.Now().Sub(start_t))
	}

	log.Println("Starting HTTP server on", options.ListenAddr)
	http.HandleFunc("/rg/", reverseGeocodingHandler)
	http.HandleFunc("/gj/", serveGeoJson)
	http.Handle("/", http.FileServer(http.Dir("html")))
	err = http.ListenAndServe(options.ListenAddr.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
}
