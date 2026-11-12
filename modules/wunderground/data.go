package wunderground

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"time"
)

// sky is the kind of icon to draw.
type sky int

const (
	skyClear sky = iota
	skyPartly
	skyCloudy
	skyRain
	skyStorm
	skySnow
	skyFog
	skyWind
)

type current struct {
	Temp, FeelsLike     float64
	Humidity            float64
	WindSpeed, WindGust float64
	WindDir             float64 // degrees, NaN if calm/unknown
	PrecipRate          float64
	PrecipToday         float64
	Pressure            float64
	UV                  float64
	Phrase              string
	Sky                 sky
	Night               bool
	Observed            time.Time
	Source              string // shown in the footer
}

type day struct {
	Date         time.Time
	High, Low    float64 // NaN when unknown (WU drops today's high in the evening)
	PrecipChance float64
	Sky          sky
	Phrase       string
}

type report struct {
	Now      current
	Days     []day
	Imperial bool
}

var httpClient = &http.Client{Timeout: 20 * time.Second}

func getJSON(ctx context.Context, u string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
		return json.NewDecoder(resp.Body).Decode(out)
	case http.StatusNoContent:
		return errors.New("no data (is the station offline?)")
	case http.StatusUnauthorized:
		return errors.New("weather underground rejected the API key")
	default:
		return fmt.Errorf("%s: %s", req.URL.Host, resp.Status)
	}
}

/* -------------------- Weather Underground -------------------- */

type wuObservation struct {
	ObsTimeUtc   string   `json:"obsTimeUtc"`
	Lat          float64  `json:"lat"`
	Lon          float64  `json:"lon"`
	Humidity     *float64 `json:"humidity"`
	Winddir      *float64 `json:"winddir"`
	UV           *float64 `json:"uv"`
	Neighborhood string   `json:"neighborhood"`
}

func fetchWU(ctx context.Context, s *Settings, loc *time.Location) (*report, error) {
	imperial := s.units != "metric"
	unit := "m"
	block := "metric"
	if imperial {
		unit, block = "e", "imperial"
	}

	q := url.Values{"stationId": {s.stationID}, "format": {"json"}, "units": {unit}, "apiKey": {s.apiKey}, "numericPrecision": {"decimal"}}
	var obs struct {
		Observations []map[string]json.RawMessage `json:"observations"`
	}
	if err := getJSON(ctx, "https://api.weather.com/v2/pws/observations/current?"+q.Encode(), &obs); err != nil {
		return nil, err
	}
	if len(obs.Observations) == 0 {
		return nil, errors.New("station returned no observation")
	}
	raw := obs.Observations[0]

	var o wuObservation
	b, _ := json.Marshal(raw)
	_ = json.Unmarshal(b, &o)
	var values map[string]*float64
	_ = json.Unmarshal(raw[block], &values)
	get := func(m map[string]*float64, k string) float64 {
		if v, ok := m[k]; ok && v != nil {
			return *v
		}
		return math.NaN()
	}

	now := current{
		Temp:        get(values, "temp"),
		Humidity:    deref(o.Humidity),
		WindSpeed:   get(values, "windSpeed"),
		WindGust:    get(values, "windGust"),
		WindDir:     deref(o.Winddir),
		PrecipRate:  get(values, "precipRate"),
		PrecipToday: get(values, "precipTotal"),
		Pressure:    get(values, "pressure"),
		UV:          deref(o.UV),
		Source:      s.stationID,
	}
	now.FeelsLike = feelsLike(now.Temp, get(values, "heatIndex"), get(values, "windChill"))
	if t, err := time.Parse(time.RFC3339, o.ObsTimeUtc); err == nil {
		now.Observed = t.In(loc)
	}

	lat, lon := s.latitude, s.longitude
	if lat == 0 && lon == 0 {
		lat, lon = o.Lat, o.Lon
	}

	days, phrase, sk, night, err := fetchWUForecast(ctx, s, lat, lon, unit, loc)
	if err != nil {
		return nil, err
	}
	now.Phrase, now.Sky, now.Night = phrase, sk, night
	if now.PrecipRate > 0 && now.Sky != skyStorm {
		now.Sky, now.Phrase = skyRain, "Raining"
	}

	return &report{Now: now, Days: days, Imperial: imperial}, nil
}

func fetchWUForecast(ctx context.Context, s *Settings, lat, lon float64, unit string, loc *time.Location) ([]day, string, sky, bool, error) {
	q := url.Values{
		"geocode":  {fmt.Sprintf("%.4f,%.4f", lat, lon)},
		"format":   {"json"},
		"units":    {unit},
		"language": {"en-US"},
		"apiKey":   {s.apiKey},
	}
	var f struct {
		ValidTimeLocal []string   `json:"validTimeLocal"`
		TemperatureMax []*float64 `json:"temperatureMax"`
		TemperatureMin []*float64 `json:"temperatureMin"`
		Daypart        []struct {
			IconCode     []*int     `json:"iconCode"`
			PrecipChance []*float64 `json:"precipChance"`
			WxPhraseLong []*string  `json:"wxPhraseLong"`
			DayOrNight   []*string  `json:"dayOrNight"`
		} `json:"daypart"`
	}
	if err := getJSON(ctx, "https://api.weather.com/v3/wx/forecast/daily/5day?"+q.Encode(), &f); err != nil {
		return nil, "", skyClear, false, err
	}
	if len(f.Daypart) == 0 {
		return nil, "", skyClear, false, errors.New("forecast had no day parts")
	}
	dp := f.Daypart[0]

	var days []day
	for i := range f.ValidTimeLocal {
		d := day{High: derefAt(f.TemperatureMax, i), Low: derefAt(f.TemperatureMin, i)}
		if t, err := time.Parse("2006-01-02T15:04:05-0700", f.ValidTimeLocal[i]); err == nil {
			d.Date = t.In(loc)
		}
		// Two day parts per day; today's daytime part is null after mid-afternoon
		for _, part := range []int{2 * i, 2*i + 1} {
			if part < len(dp.IconCode) && dp.IconCode[part] != nil {
				d.Sky = wuSky(*dp.IconCode[part])
				d.PrecipChance = derefAt(dp.PrecipChance, part)
				if part < len(dp.WxPhraseLong) && dp.WxPhraseLong[part] != nil {
					d.Phrase = *dp.WxPhraseLong[part]
				}
				break
			}
		}
		days = append(days, d)
	}

	// The first non-null day part describes right now
	phrase, sk, night := "", skyClear, false
	for p := 0; p < len(dp.IconCode) && p < 2; p++ {
		if dp.IconCode[p] == nil {
			continue
		}
		sk = wuSky(*dp.IconCode[p])
		if p < len(dp.WxPhraseLong) && dp.WxPhraseLong[p] != nil {
			phrase = *dp.WxPhraseLong[p]
		}
		night = p < len(dp.DayOrNight) && dp.DayOrNight[p] != nil && *dp.DayOrNight[p] == "N"
		break
	}

	if len(days) > s.days {
		days = days[:s.days]
	}
	return days, phrase, sk, night, nil
}

// wuSky maps The Weather Company icon codes to an icon.
func wuSky(code int) sky {
	switch {
	case code <= 4, code == 37, code == 38, code == 47:
		return skyStorm
	case code >= 5 && code <= 12, code == 35, code == 39, code == 40, code == 45:
		return skyRain
	case code >= 13 && code <= 18, code >= 41 && code <= 43, code == 46:
		return skySnow
	case code >= 19 && code <= 22:
		return skyFog
	case code == 23 || code == 24:
		return skyWind
	case code == 26 || code == 27 || code == 28:
		return skyCloudy
	case code == 29 || code == 30:
		return skyPartly
	default:
		return skyClear
	}
}

/* -------------------- Open-Meteo fallback -------------------- */

func fetchOpenMeteo(ctx context.Context, s *Settings, loc *time.Location) (*report, error) {
	if s.latitude == 0 && s.longitude == 0 {
		return nil, errors.New("add an apiKey, or latitude and longitude, to this module's config")
	}
	imperial := s.units != "metric"

	q := url.Values{
		"latitude":      {fmt.Sprint(s.latitude)},
		"longitude":     {fmt.Sprint(s.longitude)},
		"current":       {"temperature_2m,apparent_temperature,relative_humidity_2m,wind_speed_10m,wind_direction_10m,wind_gusts_10m,precipitation,pressure_msl,weather_code,is_day,uv_index"},
		"daily":         {"temperature_2m_max,temperature_2m_min,precipitation_probability_max,weather_code,precipitation_sum"},
		"timezone":      {loc.String()},
		"forecast_days": {fmt.Sprint(s.days)},
	}
	if imperial {
		q.Set("temperature_unit", "fahrenheit")
		q.Set("wind_speed_unit", "mph")
		q.Set("precipitation_unit", "inch")
	}

	var f struct {
		Current struct {
			Time     string  `json:"time"`
			Temp     float64 `json:"temperature_2m"`
			Apparent float64 `json:"apparent_temperature"`
			Humidity float64 `json:"relative_humidity_2m"`
			Wind     float64 `json:"wind_speed_10m"`
			WindDir  float64 `json:"wind_direction_10m"`
			Gust     float64 `json:"wind_gusts_10m"`
			Precip   float64 `json:"precipitation"`
			Pressure float64 `json:"pressure_msl"`
			Code     int     `json:"weather_code"`
			IsDay    int     `json:"is_day"`
			UV       float64 `json:"uv_index"`
		} `json:"current"`
		Daily struct {
			Time      []string   `json:"time"`
			Max       []*float64 `json:"temperature_2m_max"`
			Min       []*float64 `json:"temperature_2m_min"`
			Chance    []*float64 `json:"precipitation_probability_max"`
			Code      []int      `json:"weather_code"`
			PrecipSum []*float64 `json:"precipitation_sum"`
		} `json:"daily"`
	}
	if err := getJSON(ctx, "https://api.open-meteo.com/v1/forecast?"+q.Encode(), &f); err != nil {
		return nil, err
	}

	pressure := f.Current.Pressure
	if imperial {
		pressure *= 0.02953 // hPa to inHg
	}
	now := current{
		Temp:        f.Current.Temp,
		FeelsLike:   f.Current.Apparent,
		Humidity:    f.Current.Humidity,
		WindSpeed:   f.Current.Wind,
		WindGust:    f.Current.Gust,
		WindDir:     f.Current.WindDir,
		PrecipRate:  f.Current.Precip,
		PrecipToday: derefAt(f.Daily.PrecipSum, 0),
		Pressure:    pressure,
		UV:          f.Current.UV,
		Phrase:      wmoPhrase(f.Current.Code),
		Sky:         wmoSky(f.Current.Code),
		Night:       f.Current.IsDay == 0,
		Source:      "Open-Meteo",
	}
	if t, err := time.ParseInLocation("2006-01-02T15:04", f.Current.Time, loc); err == nil {
		now.Observed = t
	}

	var days []day
	for i, d := range f.Daily.Time {
		t, err := time.ParseInLocation("2006-01-02", d, loc)
		if err != nil {
			continue
		}
		code := 0
		if i < len(f.Daily.Code) {
			code = f.Daily.Code[i]
		}
		days = append(days, day{
			Date:         t,
			High:         derefAt(f.Daily.Max, i),
			Low:          derefAt(f.Daily.Min, i),
			PrecipChance: derefAt(f.Daily.Chance, i),
			Sky:          wmoSky(code),
			Phrase:       wmoPhrase(code),
		})
	}

	return &report{Now: now, Days: days, Imperial: imperial}, nil
}

func wmoSky(code int) sky {
	switch {
	case code == 0 || code == 1:
		return skyClear
	case code == 2:
		return skyPartly
	case code == 3:
		return skyCloudy
	case code == 45 || code == 48:
		return skyFog
	case code >= 71 && code <= 77, code == 85 || code == 86:
		return skySnow
	case code >= 95:
		return skyStorm
	case code >= 51:
		return skyRain
	default:
		return skyCloudy
	}
}

func wmoPhrase(code int) string {
	switch {
	case code == 0:
		return "Clear"
	case code == 1:
		return "Mostly Clear"
	case code == 2:
		return "Partly Cloudy"
	case code == 3:
		return "Cloudy"
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
		return "Snow Showers"
	case code >= 95:
		return "Thunderstorms"
	default:
		return ""
	}
}

/* -------------------- Helpers -------------------- */

// feelsLike picks the heat index when it's hotter, the wind chill when it's colder.
func feelsLike(temp, heat, chill float64) float64 {
	switch {
	case !math.IsNaN(heat) && heat > temp:
		return heat
	case !math.IsNaN(chill) && chill < temp:
		return chill
	default:
		return temp
	}
}

func deref(v *float64) float64 {
	if v == nil {
		return math.NaN()
	}
	return *v
}

func derefAt(vs []*float64, i int) float64 {
	if i < 0 || i >= len(vs) {
		return math.NaN()
	}
	return deref(vs[i])
}

func compass(deg float64) string {
	if math.IsNaN(deg) {
		return ""
	}
	dirs := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	return dirs[int(math.Mod(deg+11.25+360, 360)/22.5)%16]
}
