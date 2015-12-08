package main

import (
	"errors"
	"math"

	"github.com/dhconnelly/rtreego"
	"github.com/paulsmith/gogeos/geos"
)

var ErrNoMatchFound = errors.New("No match found in rtree")

// Searches the R-Tree to find the place corresponding to the given lat,lng
// First uses the R-Tree which contains only the bounding boxes of the
// entities, and then refine the result using the actual geometry of the entity.
func reverseGeocode(rt *rtreego.Rtree, lat, lng, precision float64) (*GeoData, error) {
	var results []rtreego.Spatial

	// First search the R-Tree
	rpt := rtreego.Point{lng, lat}
	results = rt.SearchIntersect(rpt.ToRect(precision / 2.))
	if len(results) == 0 || results[0] == nil {
		nn := rt.NearestNeighbor(rpt)
		if nn == nil {
			return nil, ErrNoMatchFound
		} else {
			results = []rtreego.Spatial{nn}
		}
	}

	// Then check with the actual geometry of the entity
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
