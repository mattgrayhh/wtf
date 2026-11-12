package todo_plus

import (
	"os"

	"github.com/olebedev/config"
	"github.com/wtfutil/wtf/cfg"
	"github.com/wtfutil/wtf/utils"
)

const (
	defaultTitle     = "Todo"
	defaultFocusable = true
)

type Settings struct {
	*cfg.Common

	backendType     string
	backendSettings *config.Config
	stacked         bool `help:"Show every project at once, one under another, instead of one at a time." optional:"true"`
}

func NewSettingsFromYAML(name string, ymlConfig *config.Config, globalConfig *config.Config) *Settings {

	backend, _ := ymlConfig.Get("backendSettings")

	settings := Settings{
		Common: cfg.NewCommonSettingsFromModule(name, defaultTitle, defaultFocusable, ymlConfig, globalConfig),

		backendType:     ymlConfig.UString("backendType"),
		backendSettings: backend,
	}

	return &settings
}

func (widget *Widget) ConfigText() string {
	return utils.HelpFromInterface(Settings{})
}

func FromTodoist(name string, ymlConfig *config.Config, globalConfig *config.Config) *Settings {
	apiKey := ymlConfig.UString("apiKey", ymlConfig.UString("apikey", os.Getenv("WTF_TODOIST_TOKEN")))
	cfg.ModuleSecret(name, globalConfig, &apiKey).Load()
	if resolved, err := utils.ResolveSecret(apiKey); err == nil {
		apiKey = resolved
	}
	projects := ymlConfig.UList("projects")
	backend, _ := config.ParseYaml("apiKey: " + apiKey)
	_ = backend.Set(".projects", projects)
	if filters, err := ymlConfig.List("filters"); err == nil {
		_ = backend.Set(".filters", filters)
	}
	if limit := ymlConfig.UInt("limit", 0); limit > 0 {
		_ = backend.Set(".limit", limit)
	}

	settings := Settings{
		Common: cfg.NewCommonSettingsFromModule(name, defaultTitle, defaultFocusable, ymlConfig, globalConfig),

		backendType:     "todoist",
		backendSettings: backend,
		stacked:         ymlConfig.UBool("stacked", false),
	}

	return &settings
}

func FromTrello(name string, ymlConfig *config.Config, globalConfig *config.Config) *Settings {

	accessToken := ymlConfig.UString("accessToken", ymlConfig.UString("apikey", os.Getenv("WTF_TRELLO_ACCESS_TOKEN")))
	apiKey := ymlConfig.UString("apiKey", os.Getenv("WTF_TRELLO_API_KEY"))
	cfg.ModuleSecret(name, globalConfig, &apiKey).Load()
	board := ymlConfig.UString("board")
	username := ymlConfig.UString("username")
	var lists []interface{}
	list, err := ymlConfig.String("list")
	if err == nil {
		lists = append(lists, list)
	} else {
		lists = ymlConfig.UList("list")
	}
	backend, _ := config.ParseYaml("apiKey: " + apiKey)
	_ = backend.Set(".accessToken", accessToken)
	_ = backend.Set(".board", board)
	_ = backend.Set(".username", username)
	_ = backend.Set(".lists", lists)

	settings := Settings{
		Common: cfg.NewCommonSettingsFromModule(name, defaultTitle, defaultFocusable, ymlConfig, globalConfig),

		backendType:     "trello",
		backendSettings: backend,
	}

	return &settings
}
