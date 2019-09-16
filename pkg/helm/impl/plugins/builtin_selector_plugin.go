package plugins

const (
	BuiltInSelectorPluginName = "BuiltInSelector"
)

func init() {
	register(BuiltInSelectorPluginName, &WalmPluginRunner{
		Run:  BuiltInSelectorTransform,
		Type: Pre_Install,
	})
}

type BuiltInSelectorArgs struct {
	ZoneSelector map[string]string `json:"zoneSelector"`
	OsSelector   map[string]string `json:"osSelector"`
	ArchSelector bool              `json:"archSelector"`
}

func BuiltInSelectorTransform(context *PluginContext, args string) (err error) {
	return
}

