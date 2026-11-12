package weatherchart

import (
	"github.com/olebedev/config"
	"github.com/wtfutil/wtf/cfg"
)

const (
	defaultFocusable = false
	defaultTitle     = "Weather"
)

// Settings defines the configuration properties for this module
type Settings struct {
	*cfg.Common

	city      string  `help:"A place name to look up, e.g. 'London' or 'Portland, Oregon'." optional:"true"`
	latitude  float64 `help:"Latitude, used instead of city." optional:"true"`
	longitude float64 `help:"Longitude, used instead of city." optional:"true"`
	units     string  `help:"Temperature units." values:"metric, imperial" optional:"true"`
	hours     int     `help:"How many hours ahead to chart. Default 24, up to 168." optional:"true"`
	textColor string  `help:"Hex colour of axis labels and legend." optional:"true"`
	fontSize  float64 `help:"Label size in points. Default 11." optional:"true"`
}

// NewSettingsFromYAML creates a new settings instance from a YAML config block
func NewSettingsFromYAML(name string, ymlConfig *config.Config, globalConfig *config.Config) *Settings {
	settings := Settings{
		Common: cfg.NewCommonSettingsFromModule(name, defaultTitle, defaultFocusable, ymlConfig, globalConfig),

		city:      ymlConfig.UString("city"),
		latitude:  ymlConfig.UFloat64("latitude"),
		longitude: ymlConfig.UFloat64("longitude"),
		units:     ymlConfig.UString("units", "metric"),
		hours:     min(max(ymlConfig.UInt("hours", 24), 6), 168),
		textColor: ymlConfig.UString("textColor"),
		fontSize:  ymlConfig.UFloat64("fontSize", 11),
	}

	return &settings
}
