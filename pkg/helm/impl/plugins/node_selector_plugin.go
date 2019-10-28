package plugins

const (
	NodeSelectorPluginName = "NodeSelector"
)

func init() {
	register(NodeSelectorPluginName, &WalmPluginRunner{
		Run:  NodeSelectorTransform,
		Type: Pre_Install,
	})
}

type BuiltInSelectorArgs struct {
	ZoneSelector map[string]string `json:"zoneSelector"`
	OsSelector   map[string]string `json:"osSelector"`
	ArchSelector bool              `json:"archSelector"`
}

func NodeSelectorTransform(context *PluginContext, args string) (err error) {
	return
}

