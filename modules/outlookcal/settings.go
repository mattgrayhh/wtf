package outlookcal

import (
	"time"

	"github.com/olebedev/config"
	"github.com/wtfutil/wtf/cfg"
)

const (
	defaultFocusable = false
	defaultTitle     = "Calendar"
)

// Settings defines the configuration properties for this module
type Settings struct {
	*cfg.Common

	account       string        `help:"The Composio connected account for Outlook (alias, word_id or ca_ id). Default is your only Outlook account." optional:"true"`
	composio      string        `help:"Path to the composio CLI. Default looks in PATH, then ~/.local/bin." optional:"true"`
	days          int           `help:"How many days to show, starting today. Default 2." optional:"true"`
	timezone      string        `help:"IANA time zone for event times. Default is local time." optional:"true"`
	showCanceled  bool          `help:"Show canceled events. Default false." optional:"true"`
	showDeclined  bool          `help:"Show events you declined. Default false." optional:"true"`
	hourFormat    string        `help:"12 or 24. Default 12." optional:"true"`
	fetchInterval time.Duration `help:"How often to fetch from Outlook, in seconds. The display refreshes every refreshInterval. Default 300." optional:"true"`

	location *time.Location
}

// NewSettingsFromYAML creates a new settings instance from a YAML config block
func NewSettingsFromYAML(name string, ymlConfig *config.Config, globalConfig *config.Config) *Settings {
	settings := Settings{
		Common: cfg.NewCommonSettingsFromModule(name, defaultTitle, defaultFocusable, ymlConfig, globalConfig),

		account:       ymlConfig.UString("account"),
		composio:      ymlConfig.UString("composio"),
		days:          min(max(ymlConfig.UInt("days", 2), 1), 14),
		timezone:      ymlConfig.UString("timezone"),
		showCanceled:  ymlConfig.UBool("showCanceled", false),
		showDeclined:  ymlConfig.UBool("showDeclined", false),
		hourFormat:    ymlConfig.UString("hourFormat", "12"),
		fetchInterval: time.Duration(ymlConfig.UInt("fetchInterval", 300)) * time.Second,

		location: time.Local,
	}

	if settings.timezone != "" {
		if loc, err := time.LoadLocation(settings.timezone); err == nil {
			settings.location = loc
		}
	}

	return &settings
}
