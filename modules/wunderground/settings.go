package wunderground

import (
	"github.com/olebedev/config"
	"github.com/wtfutil/wtf/cfg"
	"github.com/wtfutil/wtf/utils"
)

const (
	defaultFocusable = false
	defaultTitle     = "Weather"
)

// Settings defines the configuration properties for this module
type Settings struct {
	*cfg.Common

	stationID string  `help:"Your Weather Underground personal weather station ID, e.g. KGASAINT62."`
	apiKey    string  `help:"Weather Underground API key (free for station owners at wunderground.com/member/api-keys). Can be keychain:SERVICE/ACCOUNT. Without one, Open-Meteo is used at latitude/longitude." optional:"true"`
	latitude  float64 `help:"Latitude for the forecast, and for current conditions without an API key." optional:"true"`
	longitude float64 `help:"Longitude for the forecast, and for current conditions without an API key." optional:"true"`
	place     string  `help:"Name shown on the card. Default is the station ID." optional:"true"`
	units     string  `help:"imperial or metric. Default imperial." optional:"true"`
	days      int     `help:"Forecast days to show, 3 to 7. Default 5." optional:"true"`
	timezone  string  `help:"IANA time zone for times and day names. Default is local time." optional:"true"`
}

// NewSettingsFromYAML creates a new settings instance from a YAML config block
func NewSettingsFromYAML(name string, ymlConfig *config.Config, globalConfig *config.Config) *Settings {
	settings := Settings{
		Common: cfg.NewCommonSettingsFromModule(name, defaultTitle, defaultFocusable, ymlConfig, globalConfig),

		stationID: ymlConfig.UString("stationId", ymlConfig.UString("stationID")),
		apiKey:    ymlConfig.UString("apiKey"),
		latitude:  ymlConfig.UFloat64("latitude"),
		longitude: ymlConfig.UFloat64("longitude"),
		place:     ymlConfig.UString("place"),
		units:     ymlConfig.UString("units", "imperial"),
		days:      min(max(ymlConfig.UInt("days", 5), 3), 7),
		timezone:  ymlConfig.UString("timezone"),
	}

	cfg.ModuleSecret(name, globalConfig, &settings.apiKey).Load()
	if key, err := utils.ResolveSecret(settings.apiKey); err == nil {
		settings.apiKey = key
	} else {
		settings.apiKey = ""
	}

	return &settings
}
