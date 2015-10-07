package main

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strings"

	"github.com/dhconnelly/rtreego"
	"github.com/paulsmith/gogeos/geos"
)

const (
	gzipMagic  = "\x1f\x8b"
	b2zipMagic = "BZh"
)

type CountryDetails struct {
	ISO3166_3 string
	Name      string
}

// Decompressor detects the file type of an io.Reader between
// bzip, gzip, or raw file.
func Decompressor(r io.Reader) (io.Reader, error) {
	br := bufio.NewReader(r)
	buf, err := br.Peek(3)
	if err != nil {
		return nil, err
	}
	switch {
	case bytes.Compare(buf[:2], []byte(gzipMagic)) == 0:
		return gzip.NewReader(br)
	case bytes.Compare(buf[:3], []byte(b2zipMagic)) == 0:
		return bzip2.NewReader(br), nil
	default:
		return br, nil
	}
}

// Imports a CSV file mapping its ISO 3166-2 country code to its name
// The expected CSV columns are:
//   0 : ISO 3166-2 country code
//   1 : ISO 3166-3 country code
//   2 : Country name
func load_country_names(fname string) (map[string]CountryDetails, error) {
	m := make(map[string]CountryDetails)
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
		if len(cols) != 3 {
			return nil, fmt.Errorf("Invalid column format in", fname, cols)
		}
		if len(cols[0]) != 2 {
			return nil, fmt.Errorf("Invalid country code in", fname, cols)
		}
		if len(cols[1]) != 3 {
			return nil, fmt.Errorf("Invalid country code in", fname, cols)
		}
		m[cols[0]] = CountryDetails{ISO3166_3: cols[1], Name: cols[2]}
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

	df, err := Decompressor(f)
	if err != nil {
		log.Fatal(err)
	}

	var r *csv.Reader
	if strings.Contains(fname, ".tar.") {
		// Handles uncompressed files downloaded from gisgraphy.com
		tf := tar.NewReader(df)
		for {
			hdr, err := tf.Next()
			if err == io.EOF {
				log.Fatal("Couldn't find CSV file in " + fname)
			} else if err != nil {
				log.Fatal(err)
			}
			if strings.HasSuffix(hdr.Name, ".txt") {
				r = csv.NewReader(tf)
				break
			}
		}
	} else {
		// Handles unzipped files
		r = csv.NewReader(df)
	}
	r.Comma = '\t'
	line := 0
	loaded_objects := 0
	for {
		cols, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return loaded_objects, err
		}
		line++
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
				CountryCode_2: cols[3],
				Type:          "city",
				Geom:          geom}}
		rt.Insert(&obj)
		loaded_objects++
	}
	return loaded_objects, nil
}

// Loads a CSV file from https://github.com/delight-im/FreeGeoDB
// The expected CSV columns are:
// 0 : id
// 1 : coordinates_wkt
// 2 : sovereign
// 3 : name
// 4 : formal
// 5 : economy_level
// 6 : income_level
// 7 : iso_alpha2
// 8 : iso_alpha3
// 9 : iso_numeric3
// 10 : continent
// 11 : subregion
func load_freegeodb_countries_csv(rt *rtreego.Rtree, fname string) (int, error) {
	log.Println("Loading", fname, "...")

	f, err := os.Open(fname)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	df, err := Decompressor(f)
	if err != nil {
		return 0, err
	}
	r := csv.NewReader(df)
	r.Comma = ','
	_, err = r.Read() // Don't read header
	if err != nil {
		return 0, err
	}

	line := 0
	loaded_objects := 0
	for {
		cols, err := r.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return loaded_objects, err
		}
		line++

		if len(cols) != 12 {
			return loaded_objects, fmt.Errorf("Invalid column format in", fname, cols)
		}
		if len(cols[7]) != 2 || len(cols[8]) != 3 || len(cols[3]) == 0 {
			continue
		}

		geom, err := geos.FromWKT(cols[1])
		if err != nil {
			log.Fatal("Error parsing", cols, line, err)
		}
		rect, err := RtreeBboxg(geom, 1e-5)
		if err != nil {
			log.Fatal("Error getting bbox", cols, line, err)
		}
		obj := GeoObj{rect,
			&GeoData{CountryName: cols[3],
				CountryCode_2: cols[7],
				CountryCode_3: cols[8],
				Type:          "country",
				Geom:          geom}}
		rt.Insert(&obj)

		loaded_objects++
	}
	return loaded_objects, nil
}
