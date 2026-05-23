package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/yourusername/2gis-parser/internal/models"
)

const (
	// 2GIS Catalog Public API
	apiBase      = "https://catalog.api.2gis.com/3.0/items"
	pageSize     = 10
	requestDelay = 1200 * time.Millisecond // delay between requests
)

// apiResponse is the 2GIS API response shape.
type apiResponse struct {
	Result struct {
		Items []apiItem `json:"items"`
		Total int       `json:"total"`
	} `json:"result"`
	Meta struct {
		Code  int `json:"code"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"meta"`
}

type apiItem struct {
	ID              string       `json:"id"`
	Name            string       `json:"name"`
	AddressName     string       `json:"address_name"`
	FullAddressName string       `json:"full_address_name"`
	Contacts        []apiContact `json:"contact_groups"`
	Point           struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	} `json:"point"`
	Schedule map[string]interface{} `json:"schedule"`
	Rubrics  []struct {
		Name string `json:"name"`
	} `json:"rubrics"`
}

type apiContact struct {
	Contacts []struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"contacts"`
}

// Parser searches for companies through the 2GIS API.
type Parser struct {
	client   *http.Client
	apiKey   string
	maxPages int
}

func New(apiKey string, maxPages int) *Parser {
	if maxPages < 1 {
		maxPages = 1
	}
	return &Parser{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		apiKey:   apiKey,
		maxPages: maxPages,
	}
}

// Search finds companies by query in a specific city.
func (p *Parser) Search(query, city string, progressFn func(current, total int)) ([]models.Company, error) {
	regionSlug, ok := models.SupportedCities[strings.ToLower(city)]
	if !ok {
		regionSlug = strings.ToLower(city)
	}

	var all []models.Company
	page := 1

	// First request gets the total result count.
	batch, total, err := p.fetchPage(query, regionSlug, page)
	if err != nil {
		return nil, fmt.Errorf("request error: %w", err)
	}
	all = append(all, batch...)

	if progressFn != nil {
		progressFn(len(all), total)
	}

	// Pagination.
	for len(all) < total && page < p.maxPages {
		time.Sleep(requestDelay)
		page++
		batch, _, err = p.fetchPage(query, regionSlug, page)
		if err != nil {
			log.Printf("Page %d error: %v", page, err)
			break
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if progressFn != nil {
			progressFn(len(all), total)
		}
	}

	return all, nil
}

func (p *Parser) fetchPage(query, regionSlug string, page int) ([]models.Company, int, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("region_id", regionIDBySlug(regionSlug))
	params.Set("type", "branch")
	params.Set("fields", "items.address,items.contact_groups,items.point,items.schedule,items.rubrics,items.full_address_name")
	params.Set("page_size", fmt.Sprintf("%d", pageSize))
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("key", p.apiKey)

	reqURL := apiBase + "?" + params.Encode()
	log.Printf("[2GIS] GET %s", redactedURL(params))

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; B2B-Parser/1.0)")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, 0, fmt.Errorf("JSON parsing error: %w\nbody: %s", err, string(body[:min(200, len(body))]))
	}

	if apiResp.Meta.Code == http.StatusNotFound {
		return []models.Company{}, 0, nil
	}

	if apiResp.Meta.Code != 200 {
		return nil, 0, fmt.Errorf("API error %d: %s", apiResp.Meta.Code, apiResp.Meta.Error.Message)
	}

	companies := make([]models.Company, 0, len(apiResp.Result.Items))
	for _, item := range apiResp.Result.Items {
		c := models.Company{
			ID:      item.ID,
			Name:    item.Name,
			Address: firstNonEmpty(item.FullAddressName, item.AddressName),
			Lat:     item.Point.Lat,
			Lon:     item.Point.Lon,
		}
		if len(item.Rubrics) > 0 {
			c.Category = item.Rubrics[0].Name
		}
		// Phones and websites.
		for _, group := range item.Contacts {
			for _, contact := range group.Contacts {
				switch contact.Type {
				case "phone":
					if c.Phone == "" {
						c.Phone = contact.Value
					}
				case "website":
					if c.Website == "" {
						c.Website = contact.Value
					}
				}
			}
		}
		companies = append(companies, c)
	}

	return companies, apiResp.Result.Total, nil
}

func redactedURL(params url.Values) string {
	logParams := url.Values{}
	for key, values := range params {
		logParams[key] = append([]string(nil), values...)
	}
	logParams.Set("key", "***")
	return apiBase + "?" + logParams.Encode()
}

// regionIDBySlug returns the 2GIS region ID by slug.
func regionIDBySlug(slug string) string {
	ids := map[string]string{
		"astana":    "68",
		"almaty":    "67",
		"kostanay":  "203",
		"kokshetau": "201",
		"karaganda": "84",
		"petropavl": "170",
	}
	if id, ok := ids[slug]; ok {
		return id
	}
	return "68"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
