package weatherchart

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type place struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type forecast struct {
	Current struct {
		Time        string  `json:"time"`
		Temperature float64 `json:"temperature_2m"`
		WeatherCode int     `json:"weather_code"`
	} `json:"current"`
	Hourly struct {
		Time          []string   `json:"time"`
		Temperature   []*float64 `json:"temperature_2m"`
		Precipitation []*float64 `json:"precipitation_probability"`
	} `json:"hourly"`
	Daily struct {
		Max []float64 `json:"temperature_2m_max"`
		Min []float64 `json:"temperature_2m_min"`
	} `json:"daily"`
}

func geocode(ctx context.Context, name string) (place, error) {
	// Open-Meteo matches the place name only, so drop any ", region" suffix
	// for the lookup but keep it for the title.
	query := strings.TrimSpace(strings.Split(name, ",")[0])
	u := "https://geocoding-api.open-meteo.com/v1/search?count=1&name=" + url.QueryEscape(query)

	var res struct {
		Results []place `json:"results"`
	}
	if err := getJSON(ctx, u, &res); err != nil {
		return place{}, err
	}
	if len(res.Results) == 0 {
		return place{}, fmt.Errorf("no place called %q", name)
	}
	return res.Results[0], nil
}

func fetchForecast(ctx context.Context, lat, lon float64, imperial bool, hours int) (*forecast, error) {
	q := url.Values{}
	q.Set("latitude", fmt.Sprint(lat))
	q.Set("longitude", fmt.Sprint(lon))
	q.Set("current", "temperature_2m,weather_code")
	q.Set("hourly", "temperature_2m,precipitation_probability")
	q.Set("daily", "temperature_2m_max,temperature_2m_min")
	q.Set("timezone", "auto")
	q.Set("forecast_days", fmt.Sprint(hours/24+2))
	if imperial {
		q.Set("temperature_unit", "fahrenheit")
	}

	var f forecast
	if err := getJSON(ctx, "https://api.open-meteo.com/v1/forecast?"+q.Encode(), &f); err != nil {
		return nil, err
	}
	if len(f.Hourly.Time) == 0 {
		return nil, errors.New("forecast had no hourly data")
	}
	return &f, nil
}

func getJSON(ctx context.Context, u string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("open-meteo: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// conditions turns a WMO weather code into words.
func conditions(code int) string {
	switch {
	case code == 0:
		return "Clear"
	case code == 1:
		return "Mostly clear"
	case code == 2:
		return "Partly cloudy"
	case code == 3:
		return "Overcast"
	case code == 45 || code == 48:
		return "Fog"
	case code >= 51 && code <= 57:
		return "Drizzle"
	case code >= 61 && code <= 67:
		return "Rain"
	case code >= 71 && code <= 77:
		return "Snow"
	case code >= 80 && code <= 82:
		return "Showers"
	case code == 85 || code == 86:
		return "Snow showers"
	case code >= 95:
		return "Thunderstorm"
	default:
		return ""
	}
}
