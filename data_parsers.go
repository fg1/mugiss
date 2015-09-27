package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"os"

	"github.com/dhconnelly/rtreego"
	"github.com/paulsmith/gogeos/geos"
)

// Imports a CSV file mapping its ISO 3166-2 country code to its name
// The expected CSV columns are:
//   0 : ISO 3166-2 country code
//   1 : Country name
func load_country_names(fname string) (map[string]string, error) {
	m := make(map[string]string)
	f, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	for {
		cols, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(cols) != 2 {
			return nil, fmt.Errorf("Invalid column format in", fname, cols)
		}
		if len(cols[0]) != 2 {
			return nil, fmt.Errorf("Invalid country code in", fname, cols)
		}
		m[cols[0]] = cols[1]
	}
	return m, nil
}

// Returns the bounding box of the geos.Geometry as a rtreego.Rect
func RtreeBboxg(g *geos.Geometry, tol float64) (*rtreego.Rect, error) {
	env, err := g.Envelope()
	if err != nil {
		return nil, err
	}
	shell, _ := env.Shell()
	if err != nil {
		return nil, err
	}
	c, err := shell.Coords()
	if err != nil {
		return nil, err
	}
	return rtreego.NewRect(rtreego.Point{c[0].X, c[0].Y},
		[]float64{math.Max(tol, c[2].X-c[0].X),
			math.Max(tol, c[2].Y-c[0].Y)})
}

// Loads a CSV file from http://download.gisgraphy.com/openstreetmap/csv/cities/
// The expected CSV columns are:
//   0 :  Node type;   N|W|R (in uppercase), wheter it is a Node, Way or Relation in the openstreetmap Model
//   1 :  id;  The openstreetmap id
//   2 :  name;    the default name of the city
//   3 :  countrycode; The iso3166-2 country code (2 letters)
//   4 :  postcode;    The postcode / zipcode / ons code / municipality code / ...
//   5 :  population;  How many people lives in that city
//   6 :  location;    The middle location of the city in HEXEWKB
//   7 :  shape; The delimitation of the city in HEXEWKB
//   8 :  type; the type of city ('city', 'village', 'town', 'hamlet', ...)
//   9 :  is_in ; where the cities is located (generally the fully qualified administrative division)
//   10 : alternatenames;     the names of the city in other languages
func load_gisgraphy_cities_csv(rt *rtreego.Rtree, fname string) (int, error) {
	log.Println("Loading", fname, "...")
	f, err := os.Open(fname)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	line := 0
	loaded_cities := 0
	r := csv.NewReader(f)
	r.Comma = '\t'
	for {
		cols, err := r.Read()
		line++
		if err == io.EOF {
			break
		}
		if err != nil {
			return loaded_cities, err
		}
		if len(cols[7]) == 0 {
			continue
		}

		geom, err := geos.FromHex(cols[7])
		if err != nil {
			log.Fatal("Error parsing", cols, line, err)
		}
		rect, err := RtreeBboxg(geom, 1e-5)
		if err != nil {
			log.Fatal("Error getting bbox", cols, line, err)
		}
		obj := GeoObj{rect,
			&GeoData{City: cols[2],
				CountryCode: cols[3],
				Type:        cols[8],
				Geom:        geom}}
		rt.Insert(&obj)
		loaded_cities++
	}
	return loaded_cities, nil
}
