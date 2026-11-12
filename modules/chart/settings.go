package chart

import (
	"github.com/olebedev/config"
	"github.com/wtfutil/wtf/cfg"
	"github.com/wtfutil/wtf/utils"
)

const (
	defaultFocusable = false
	defaultTitle     = "Chart"
)

// Settings defines the configuration properties for this module
type Settings struct {
	*cfg.Common

	cmd       string   `help:"A command that prints the data. See the module docs for the format." optional:"true"`
	args      []string `help:"Arguments to cmd, one per array item." optional:"true"`
	path      string   `help:"A CSV or TSV file holding the data." optional:"true"`
	url       string   `help:"A URL returning CSV or TSV data." optional:"true"`
	kind      string   `help:"How series are drawn." values:"line, area, bar" optional:"true"`
	colors    []string `help:"Hex colours for each series, in order." optional:"true"`
	secondary []string `help:"Names of series plotted against a second axis on the right." optional:"true"`
	legend    string   `help:"Show the legend: auto (when there are several series), true or false." optional:"true"`
	maxPoints int      `help:"Only chart the last N rows. 0 charts everything." optional:"true"`
	yMin      *float64 `help:"Fix the bottom of the y axis." optional:"true"`
	yMax      *float64 `help:"Fix the top of the y axis." optional:"true"`
	yFormat   string   `help:"Format for y axis labels, e.g. '%.0f°' or '$%.2f'." optional:"true"`
	textColor string   `help:"Hex colour of axis labels and legend." optional:"true"`
	fontSize  float64  `help:"Label size in points. Default 11." optional:"true"`
}

// NewSettingsFromYAML creates a new settings instance from a YAML config block
func NewSettingsFromYAML(name string, ymlConfig *config.Config, globalConfig *config.Config) *Settings {
	settings := Settings{
		Common: cfg.NewCommonSettingsFromModule(name, defaultTitle, defaultFocusable, ymlConfig, globalConfig),

		cmd:       ymlConfig.UString("cmd"),
		args:      utils.ToStrs(ymlConfig.UList("args")),
		path:      ymlConfig.UString("path"),
		url:       ymlConfig.UString("url"),
		kind:      ymlConfig.UString("kind", "line"),
		colors:    utils.ToStrs(ymlConfig.UList("colors")),
		secondary: utils.ToStrs(ymlConfig.UList("secondary")),
		legend:    ymlConfig.UString("legend", "auto"),
		maxPoints: ymlConfig.UInt("maxPoints", 0),
		yMin:      optionalFloat(ymlConfig, "yMin"),
		yMax:      optionalFloat(ymlConfig, "yMax"),
		yFormat:   ymlConfig.UString("yFormat"),
		textColor: ymlConfig.UString("textColor"),
		fontSize:  ymlConfig.UFloat64("fontSize", 11),
	}

	return &settings
}

func optionalFloat(c *config.Config, key string) *float64 {
	v, err := c.Float64(key)
	if err != nil {
		return nil
	}
	return &v
}
