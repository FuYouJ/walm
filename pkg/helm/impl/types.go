package impl

type TranswarpAppInfo struct {
	AppDependency
	UserInputParams *UserInputParams `json:"userInputParams"`
}

type UserInputParams struct {
	CommonConfig        CommonConfig  `json:"commonConfig"`
	TranswarpBaseConfig []*BaseConfig `json:"transwarpBundleConfig"`
	AdvanceConfig       []*BaseConfig `json:"advanceConfig"`
}

type ResourceStorageConfig struct {
	Name         string   `json:"name"`
	StorageType  string   `json:"type"`
	StorageClass string   `json:"storageClass"`
	Size         string   `json:"size"`
	AccessModes  []string `json:"accessModes"`
	AccessMode   string   `json:"accessMode"`
	DiskReplicas int      `json:"disk_replicas"`
}

type ResourceConfig struct {
	CpuLimit            float64                 `json:"cpu_limit"`
	CpuRequest          float64                 `json:"cpu_request"`
	MemoryLimit         float64                 `json:"memory_limit"`
	MemoryRequest       float64                 `json:"memory_request"`
	GpuLimit            int                     `json:"gpu_limit"`
	GpuRequest          int                     `json:"gpu_request"`
	ResourceStorageList []ResourceStorageConfig `json:"storage"`
}

type BaseConfig struct {
	ValueName        string      `json:"variable" description:"variable name"`
	DefaultValue     interface{} `json:"default" description:"variable default value"`
	ValueDescription string      `json:"description" description:"variable description"`
	ValueType        string      `json:"type" description:"variable type"`
}

type RoleConfig struct {
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	Replicas           int             `json:"replicas"`
	RoleBaseConfig     []*BaseConfig   `json:"baseConfig"`
	RoleResourceConfig *ResourceConfig `json:"resouceConfig"`
}

type CommonConfig struct {
	Roles []*RoleConfig `json:"roles"`
}

type DependencyDeclare struct {
	// name of dependency declaration
	Name string `json:"name,omitempty"`
	// dependency variable mappings
	Requires map[string]string `json:"requires,omitempty"`
	// minVersion
	MinVersion float32 `json:"minVersion"`
	// maxVersion
	MaxVersion float32 `json:"maxVersion"`

	DependencyOptional bool `json:"dependencyOptional"`
}

type AppDependency struct {
	Name string `json:"name,omitempty"`
	Dependencies []*DependencyDeclare `json:"dependencies"`
}
