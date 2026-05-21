// Package location provides geolocation detection via IP lookup.
package location

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/httpclient"
	tools "github.com/oneliang/aura/tools/pkg"
)

// API endpoints for IP-based geolocation.
const (
	ipinfoURL = "https://ipinfo.io/json"
	ipapiURL  = "http://ip-api.com/json/"
	cacheTTL  = 24 * time.Hour
)

// LocationConfig holds configuration overrides for the location tool.
type LocationConfig struct {
	FixedCity    string // Fixed city name (overrides auto-detection)
	FixedCountry string // Fixed country name
	AutoDetect   bool   // Enable IP-based auto-detection
}

// LocationData holds resolved location information.
type LocationData struct {
	City        string  `json:"city"`
	Region      string  `json:"region"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Source      string  `json:"source"`
}

// LocationTool provides user geolocation via IP lookup.
type LocationTool struct {
	client *http.Client
	cfg    LocationConfig

	mu     sync.RWMutex
	cache  *LocationData
	expiry time.Time
}

// LocationOption configures a LocationTool.
type LocationOption func(*LocationTool)

// WithHTTPClient sets the HTTP client.
func WithHTTPClient(client *http.Client) LocationOption {
	return func(t *LocationTool) {
		t.client = client
	}
}

// WithConfig sets the configuration overrides.
func WithConfig(cfg LocationConfig) LocationOption {
	return func(t *LocationTool) {
		t.cfg = cfg
	}
}

// NewLocationTool creates a new location tool.
func NewLocationTool(opts ...LocationOption) *LocationTool {
	t := &LocationTool{
		client: httpclient.DefaultWebClient(),
	}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Name returns the tool name.
func (t *LocationTool) Name() string {
	return constants.ToolLocation
}

// Description returns the tool description.
func (t *LocationTool) Description() string {
	return "Get user's current location via IP geolocation. Returns city, region, country, and coordinates. No parameters required."
}

// Execute returns location information.
func (t *LocationTool) Execute(ctx context.Context, params map[string]any) (*tools.ToolResult, error) {
	loc, err := t.getLocation(ctx)
	if err != nil {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: err.Error()}, nil
	}

	return &tools.ToolResult{
		Status: tools.ToolStatusSuccess,
		Content: fmt.Sprintf(`Location:
  City: %s
  Region: %s
  Country: %s (%s)
  Latitude: %.4f
  Longitude: %.4f
  Source: %s`,
			loc.City, loc.Region, loc.Country, loc.CountryCode,
			loc.Lat, loc.Lon, loc.Source,
		),
	}, nil
}

// getLocation returns the user's location, using cache/config/IP lookup.
func (t *LocationTool) getLocation(ctx context.Context) (*LocationData, error) {
	// Config override takes highest priority
	if t.cfg.FixedCity != "" {
		return &LocationData{
			City:        t.cfg.FixedCity,
			Region:      "",
			Country:     t.cfg.FixedCountry,
			CountryCode: "",
			Source:      "config",
		}, nil
	}

	// Check cache
	t.mu.RLock()
	if t.cache != nil && time.Now().Before(t.expiry) {
		cached := *t.cache
		t.mu.RUnlock()
		return &cached, nil
	}
	t.mu.RUnlock()

	// Auto-detect via IP
	if !t.cfg.AutoDetect {
		return nil, fmt.Errorf("location auto-detection is disabled, set auto_detect: true in config or specify fixed_city")
	}

	loc, err := t.detectViaIP(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to detect location: %w", err)
	}

	// Update cache
	t.mu.Lock()
	t.cache = loc
	t.expiry = time.Now().Add(cacheTTL)
	t.mu.Unlock()

	return loc, nil
}

// detectViaIP attempts IP-based geolocation with primary/backup APIs.
func (t *LocationTool) detectViaIP(ctx context.Context) (*LocationData, error) {
	// Try primary: ipinfo.io
	if loc, err := t.fetchIPInfo(ctx); err == nil && loc.City != "" {
		loc.Source = "ipinfo"
		return loc, nil
	}

	// Fallback: ip-api.com
	if loc, err := t.fetchIPAPI(ctx); err == nil && loc.City != "" {
		loc.Source = "ip-api"
		return loc, nil
	}

	return nil, fmt.Errorf("all IP geolocation APIs failed")
}

// ipinfoResponse matches ipinfo.io JSON response.
type ipinfoResponse struct {
	City    string `json:"city"`
	Region  string `json:"region"`
	Country string `json:"country"`
	Loc     string `json:"loc"`
}

func (t *LocationTool) fetchIPInfo(ctx context.Context) (*LocationData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ipinfoURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ipinfo API error: %d", resp.StatusCode)
	}

	var body ipinfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("ipinfo JSON decode: %w", err)
	}

	data := &LocationData{
		City:    body.City,
		Region:  body.Region,
		Country: body.Country,
	}

	// Parse "lat,lon" format
	if body.Loc != "" {
		parts := strings.SplitN(body.Loc, ",", 2)
		if len(parts) == 2 {
			if lat, err := strconv.ParseFloat(parts[0], 64); err == nil {
				data.Lat = lat
			}
			if lon, err := strconv.ParseFloat(parts[1], 64); err == nil {
				data.Lon = lon
			}
		}
	}

	return data, nil
}

// ipapiResponse matches ip-api.com JSON response.
type ipapiResponse struct {
	City    string  `json:"city"`
	Region  string  `json:"regionName"`
	Country string  `json:"country"`
	Code    string  `json:"countryCode"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Status  string  `json:"status"`
}

func (t *LocationTool) fetchIPAPI(ctx context.Context) (*LocationData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ipapiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ip-api API error: %d", resp.StatusCode)
	}

	var body ipapiResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("ip-api JSON decode: %w", err)
	}

	if body.Status == "fail" {
		return nil, fmt.Errorf("ip-api returned fail status")
	}

	return &LocationData{
		City:        body.City,
		Region:      body.Region,
		Country:     body.Country,
		CountryCode: body.Code,
		Lat:         body.Lat,
		Lon:         body.Lon,
	}, nil
}
