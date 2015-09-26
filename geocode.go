package main

import (
	"fmt"

	"github.com/dhconnelly/rtreego"
	"github.com/paulsmith/gogeos/geos"
)

// Searches the R-Tree to find the place corresponding to the given lat,lng
// First do a search of the nearest entities (city, ...) according to their
// bounding box, and then refine it using the actual geometry of the entity.
func reverseGeocode(rt *rtreego.Rtree, lat, lng float64) (*GeoData, error) {
	rpt := rtreego.Point{lng, lat}

	// TODO: Check that 10 is a good value or modify rtreego
	results := rt.NearestNeighbors(10, rpt)
	if results[0] == nil {
		return nil, fmt.Errorf("No result returned by RTree")
	}

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
	}
	return nil, fmt.Errorf("No containing city found")
}
