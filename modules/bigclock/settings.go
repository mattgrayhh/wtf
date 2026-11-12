package bigclock

import (
	"time"

	"github.com/olebedev/config"
	"github.com/wtfutil/wtf/cfg"
)

const (
	defaultFocusable = false
	defaultTitle     = "Clock"
)

// Settings defines the configuration properties for this module
type Settings struct {
	*cfg.Common

	timezone   string `help:"IANA time zone, e.g. America/New_York. Default is local time." optional:"true"`
	hourFormat string `help:"12 or 24. Default 12." optional:"true"`
	dateFormat string `help:"Go layout for the date line. Default 'Monday, January 2'." optional:"true"`
	color      string `help:"Hex colour of the time. Default near-white." optional:"true"`

	location *time.Location
}

// NewSettingsFromYAML creates a new settings instance from a YAML config block
func NewSettingsFromYAML(name string, ymlConfig *config.Config, globalConfig *config.Config) *Settings {
	settings := Settings{
		Common: cfg.NewCommonSettingsFromModule(name, defaultTitle, defaultFocusable, ymlConfig, globalConfig),

		timezone:   ymlConfig.UString("timezone"),
		hourFormat: ymlConfig.UString("hourFormat", "12"),
		dateFormat: ymlConfig.UString("dateFormat", "Monday, January 2"),
		color:      ymlConfig.UString("color", "#eef0f4"),

		location: time.Local,
	}
	if settings.timezone != "" {
		if loc, err := time.LoadLocation(settings.timezone); err == nil {
			settings.location = loc
		}
	}
	return &settings
}
