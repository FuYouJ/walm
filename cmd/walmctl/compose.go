package main

import (
	"bytes"
	"github.com/Masterminds/sprig"
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"k8s.io/klog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"WarpCloud/walm/cmd/walmctl/util/walmctlclient"
)

const composeDesc = `
Compose a Walm Compose file
`

type composeCmd struct {
	projectName string
	file   string
	dryrun bool
}

func newComposeCmd() *cobra.Command {
	compose := composeCmd{}
	cmd := &cobra.Command{
		Use:   "compose [file]",
		Short: "Compose a Walm Compose file",
		Long:  composeDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if walmserver == "" {
				return errServerRequired
			}
			if namespace == "" {
				return errNamespaceRequired
			}
			return compose.run()
		},
	}
	cmd.PersistentFlags().StringVar(&compose.file, "file", "compose.yaml", "walm compose file")
	cmd.Flags().BoolVar(&compose.dryrun, "dryrun", false, "dry run")
	cmd.Flags().StringVarP(&compose.projectName, "project", "p", "", "project name")
	cmd.MarkFlagRequired("file")
	cmd.MarkFlagRequired("project")

	return cmd
}

func (compose *composeCmd) run() error {
	client := walmctlclient.CreateNewClient(walmserver)
	if err := client.ValidateHostConnect(); err != nil {
		return err
	}

	var t *template.Template
	filePath, err := filepath.Abs(compose.file)
	if err != nil {
		return err
	}
	t, err = parseFiles(filePath)
	if err != nil {
		return err
	}
	env := readEnv()
	var fileTpl bytes.Buffer
	env["NAMESPACE"] = namespace
	env["PROJECT_NAME"] = compose.projectName
	err = t.Execute(&fileTpl, env)
	configValues := make(map[string]interface{}, 0)
	err = yaml.Unmarshal(fileTpl.Bytes(), &configValues)
	if err != nil {
		klog.Errorf("yaml Unmarshal file %s error %v", compose.file, err)
		return err
	}
	_, err = client.CreateProject(namespace, "", compose.projectName, false, 300, configValues)
	return err
}

// returns map of environment variables
func readEnv() (env map[string]string) {
	env = make(map[string]string)
	for _, setting := range os.Environ() {
		pair := strings.SplitN(setting, "=", 2)
		env[pair[0]] = pair[1]
	}
	return
}

func parseFiles(files ...string) (*template.Template, error) {
	return template.New(filepath.Base(files[0])).Funcs(sprig.TxtFuncMap()).Funcs(customFuncMap()).ParseFiles(files...)
}

// custom function that returns key, value for all environment variable keys matching prefix
// (see original envtpl: https://pypi.org/project/envtpl/)
func environment(prefix string) map[string]string {
	env := make(map[string]string)
	for _, setting := range os.Environ() {
		pair := strings.SplitN(setting, "=", 2)
		if strings.HasPrefix(pair[0], prefix) {
			env[pair[0]] = pair[1]
		}
	}
	return env
}

// returns custom template functions map
func customFuncMap() template.FuncMap {
	var functionMap = map[string]interface{}{
		"environment": environment,
	}
	return template.FuncMap(functionMap)
}
