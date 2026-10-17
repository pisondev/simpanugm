package geo

import "math"

const earthRadiusM = 6371000.0

func HaversineM(lat1, lng1, lat2, lng2 float64) float64 {
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLng := (lng2 - lng1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLng/2)*math.Sin(dLng/2)
	return 2 * earthRadiusM * math.Asin(math.Min(1, math.Sqrt(a)))
}

type Box struct {
	MinLat float64 `yaml:"min_lat" json:"min_lat"`
	MaxLat float64 `yaml:"max_lat" json:"max_lat"`
	MinLng float64 `yaml:"min_lng" json:"min_lng"`
	MaxLng float64 `yaml:"max_lng" json:"max_lng"`
}

func (b Box) Contains(lat, lng float64) bool {
	return lat >= b.MinLat && lat <= b.MaxLat && lng >= b.MinLng && lng <= b.MaxLng
}
