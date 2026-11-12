package image

import (
	"github.com/olebedev/config"
	"github.com/wtfutil/wtf/cfg"
	"github.com/wtfutil/wtf/utils"
)

const (
	defaultFocusable = false
	defaultTitle     = "Image"
)

// Settings defines the configuration properties for this module
type Settings struct {
	*cfg.Common

	path string   `help:"A local image file (PNG, JPEG, GIF or WebP)." optional:"true"`
	url  string   `help:"An image URL, fetched on every refresh. Example: https://wttr.in/London.png" optional:"true"`
	cmd  string   `help:"A command whose output is the image itself, or a path or URL to it. Useful for album art." optional:"true"`
	args []string `help:"Arguments to cmd, one per array item." optional:"true"`
	fit  string   `help:"How the image fills the panel: 'contain' keeps its shape, 'fill' stretches it." values:"contain, fill" optional:"true"`
}

// NewSettingsFromYAML creates a new settings instance from a YAML config block
func NewSettingsFromYAML(name string, ymlConfig *config.Config, globalConfig *config.Config) *Settings {
	settings := Settings{
		Common: cfg.NewCommonSettingsFromModule(name, defaultTitle, defaultFocusable, ymlConfig, globalConfig),

		path: ymlConfig.UString("path"),
		url:  ymlConfig.UString("url"),
		cmd:  ymlConfig.UString("cmd"),
		args: utils.ToStrs(ymlConfig.UList("args")),
		fit:  ymlConfig.UString("fit", "contain"),
	}

	return &settings
}
