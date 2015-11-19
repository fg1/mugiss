package main

import (
	"github.com/paulsmith/gogeos/geos"
)

type gjFeatureCollection struct {
	Type     string      `json:"type"`
	Features []gjFeature `json:"features"`
}

type gjFeature struct {
	Type       string     `json:"type"`
	Geometry   gjGeometry `json:"geometry"`
	Properties GeoData    `json:"properties"`
}

// Note: gjGeometry is limited to the MultiPolygon primitive
type gjGeometry struct {
	Type   string          `json:"type"`
	Coords [][][][]float64 `json:"coordinates"`
}

func ToGjFeature(g *geos.Geometry) (gjFeature, error) {
	var gjf gjFeature
	gjf.Type = "Feature"
	gjf.Geometry.Type = "MultiPolygon"
	ngeoms, _ := g.NGeometry()
	gjf.Geometry.Coords = make([][][][]float64, ngeoms)

	for i := 0; i < ngeoms; i++ {
		subg, err := g.Geometry(i)
		if err != nil {
			return gjf, err
		}
		holes, err := subg.Holes()
		if err != nil {
			return gjf, err
		}
		gjf.Geometry.Coords[i] = make([][][]float64, 1+len(holes))

		shell, err := subg.Shell()
		if err != nil {
			return gjf, err
		}

		coords, err := shell.Coords()
		if err != nil {
			return gjf, err
		}
		gjf.Geometry.Coords[i][0] = make([][]float64, len(coords))
		for j, coord := range coords {
			gjf.Geometry.Coords[i][0][j] = make([]float64, 2)
			gjf.Geometry.Coords[i][0][j][0] = coord.X
			gjf.Geometry.Coords[i][0][j][1] = coord.Y
		}

		for k, hole := range holes {
			coords, err := hole.Coords()
			if err != nil {
				return gjf, err
			}

			gjf.Geometry.Coords[i][1+k] = make([][]float64, len(coords))
			for j, coord := range coords {
				gjf.Geometry.Coords[i][1+k][j] = make([]float64, 2)
				gjf.Geometry.Coords[i][1+k][j][0] = coord.X
				gjf.Geometry.Coords[i][1+k][j][1] = coord.Y
			}
		}
	}
	return gjf, nil
}
