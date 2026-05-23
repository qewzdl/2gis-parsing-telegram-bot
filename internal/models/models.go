package models

// Company represents company data from 2GIS.
type Company struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Address      string  `json:"address"`
	City         string  `json:"city"`
	Phone        string  `json:"phone"`
	Website      string  `json:"website"`
	WorkingHours string  `json:"working_hours"`
	Category     string  `json:"category"`
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
}

// SearchRequest is a user search request.
type SearchRequest struct {
	UserID int64
	Query  string
	Cities []string
}

// ParseResult contains parsing results.
type ParseResult struct {
	City      string
	Query     string
	Companies []Company
	Total     int
	Error     error
}

// SupportedCities maps supported Kazakhstan city names to 2GIS region slugs.
var SupportedCities = map[string]string{
	"astana":        "astana",
	"almaty":        "almaty",
	"kostanay":      "kostanay",
	"kokshetau":     "kokshetau",
	"kokchetav":     "kokshetau",
	"karaganda":     "karaganda",
	"petropavlovsk": "petropavl",
	"petropavl":     "petropavl",
}
