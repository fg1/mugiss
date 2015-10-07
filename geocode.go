package main

import (
	"errors"
	"math"

	"github.com/dhconnelly/rtreego"
	"github.com/paulsmith/gogeos/geos"
)

var ErrNoMatchFound = errors.New("No match found in rtree")

// Searches the R-Tree to find the place corresponding to the given lat,lng
// First do a search of the nearest entities (city, ...) according to their
// bounding box, and then refine it using the actual geometry of the entity.
func reverseGeocode(rt *rtreego.Rtree, lat, lng, precision float64) (*GeoData, error) {
	rpt := rtreego.Point{lng, lat}

	// TODO: Check that 10 is a good value or modify rtreego
	results := rt.NearestNeighbors(10, rpt)
	if results[0] == nil {
		return nil, ErrNoMatchFound
	}

	mindist := math.MaxFloat64
	var argmindist *GeoData
	gpt, _ := geos.NewPoint(geos.NewCoord(lng, lat))
	for _, res := range results {
		if res == nil {
			break
		}
		obj, _ := res.(SpatialData)
		geod := obj.GetData()
		inside, _ := geod.Geom.Contains(gpt)
		if inside {
			return geod, nil
		}

		dist, _ := gpt.Distance(geod.Geom)
		if dist < mindist {
			mindist = dist
			argmindist = geod
		}
	}

	if mindist < precision {
		return argmindist, nil
	}
	return nil, ErrNoMatchFound
}
