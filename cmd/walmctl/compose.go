package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Masterminds/sprig"
	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"k8s.io/klog"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"WarpCloud/walm/cmd/walmctl/util/guardianclient"
	"WarpCloud/walm/cmd/walmctl/util/walmctlclient"
	"WarpCloud/walm/pkg/models/project"
	"WarpCloud/walm/pkg/models/release"
)

const composeDesc = `
Compose a Walm Compose file
`

type composeCmd struct {
	projectName string
	file        string
	dryrun      bool
	waitReady   bool
	walmClient  *walmctlclient.WalmctlClient
}

type composeGuardianConfig struct {
	SecretName  string   `json:"secretName"`
	GuardianURL string   `json:"guardianURL"`
	User        string   `json:"user"`
	Password    string   `json:"password"`
	Principals  []string `json:"principals"`
	Krb5Conf    string   `json:"krb5.conf"`
}

type composeFunc struct {
	FuncName string `json:"name"`
	Args     string `json:"args"`
	Output   string `json:"output"`
}

type composeFuncsConfig struct {
	Functions []composeFunc `json:"functions"`
}

type composeConfig struct {
	ProjectConfigs  map[string]interface{}   `json:"projectConfigs"`
	GuardianConfigs []*composeGuardianConfig `json:"guardianConfigs"`
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
	cmd.PersistentFlags().StringVarP(&compose.file, "file", "f", "compose.yaml", "walm compose file")
	cmd.Flags().BoolVar(&compose.dryrun, "dryrun", false, "dry run")
	cmd.Flags().StringVarP(&compose.projectName, "project", "p", "", "project name")
	cmd.Flags().BoolVarP(&compose.waitReady, "waitready", "w", false, "wait project ready")
	cmd.MarkFlagRequired("file")
	cmd.MarkFlagRequired("project")

	return cmd
}

func (compose *composeCmd) run() error {
	client := walmctlclient.CreateNewClient(walmserver)
	if err := client.ValidateHostConnect(); err != nil {
		return err
	}
	compose.walmClient = client
	filePath, err := filepath.Abs(compose.file)
	if err != nil {
		return err
	}
	env := readEnv()

	err = compose.expandComposeFuncConfigs(filePath, env)
	if err != nil {
		return err
	}

	var t *template.Template
	t, err = parseFiles(filePath)
	if err != nil {
		return err
	}
	var fileTpl bytes.Buffer
	err = t.Execute(&fileTpl, env)
	configValues := composeConfig{}
	err = yaml.Unmarshal(fileTpl.Bytes(), &configValues)
	if err != nil {
		klog.Errorf("yaml Unmarshal file %s error %v", compose.file, err)
		return err
	}

	_, err = client.CreateProject(namespace, "", compose.projectName, false, 300, configValues.ProjectConfigs)
	_ = compose.generateGuardianKeytabSecrets(configValues.GuardianConfigs)

	return err
}

func (compose *composeCmd) expandComposeFuncConfigs(filePath string, envMap map[string]string) error {
	envMap["NAMESPACE"] = namespace
	envMap["PROJECT_NAME"] = compose.projectName
	if _, ok := envMap["CLUSTER_HOST"]; !ok {
		envMap["CLUSTER_HOST"] = strings.Split(walmserver, ":")[0]
	}

	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		klog.Errorf("read file %s error %v", compose.file, err)
		return err
	}
	configFuncs := composeFuncsConfig{}
	err = yaml.Unmarshal(fileBytes, &configFuncs)
	if err != nil {
		klog.Errorf("yaml unmarshal file %s error %v", compose.file, err)
		return err
	}
	for _, funcConfig := range configFuncs.Functions {
		if funcConfig.FuncName == "funcGetInstallId" {
			installId, err := compose.funcGetInstallId(funcConfig.Args)
			if err != nil {
				return err
			}
			envMap[funcConfig.Output] = installId
			klog.Infof("set %s=%s", funcConfig.Output, installId)
		}
	}

	return nil
}

func (compose *composeCmd) funcGetInstallId(releaseName string) (string, error) {
	releaseInfo := release.ReleaseInfo{}
	resp, err := compose.walmClient.GetRelease(namespace, releaseName)
	if err != nil {
		klog.Errorf("get release %s/%s error %v", namespace, releaseName, err)
		return "", err
	}
	err = json.Unmarshal(resp.Body(), &releaseInfo)
	if err != nil {
		klog.Errorf("json unmarshal error %v", err)
		return "", err
	}
	installId, ok := releaseInfo.ReleaseSpec.ConfigValues["Transwarp_Install_ID"]
	if !ok {
		klog.Error("Transwarp_Install_ID not found")
		return "", errors.New("Transwarp_Install_ID not found")
	}

	return installId.(string), err
}

func (compose *composeCmd) generateGuardianKeytabSecrets(guardianConfigs []*composeGuardianConfig) bool {
	ch := make(chan bool, 1)
	for _, guardianConfig := range guardianConfigs {
		go func(config *composeGuardianConfig) {
			ready := make(chan bool)
			go compose.retryCreateGuardianKeytabUtilTimeout(config, ready)
			select {
			case ret := <-ready:
				ch <- ret
			case <-time.After(600 * time.Second):
				ch <- false
			}
		}(guardianConfig)
	}

	return <-ch
}

func (compose *composeCmd) retryCreateGuardianKeytabUtilTimeout(guardianConfig *composeGuardianConfig, ready chan<- bool) {
	gClient := guardianclient.NewClient(guardianConfig.GuardianURL, guardianConfig.User, guardianConfig.Password)
	for true {
		keytabData, err := gClient.GetMultipleKeytabs(guardianConfig.Principals)
		if err != nil {
			klog.Errorf("get Guardian Keytab error %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		secretData := make(map[string]string, 0)
		secretData["keytab"] = base64.StdEncoding.EncodeToString(keytabData)
		secretData["krb5.conf"] = base64.StdEncoding.EncodeToString([]byte(guardianConfig.Krb5Conf))

		_ = compose.walmClient.DeleteSecret(namespace, guardianConfig.SecretName)
		err = compose.walmClient.CreateSecret(namespace, guardianConfig.SecretName, secretData)
		if err != nil {
			klog.Errorf("create secret %v", err)
		}
		break
	}
	ready <- true
}

func (compose *composeCmd) waitProjectReady(timeout time.Duration) error {
	startTime := time.Now()
	for true {
		resp, err := compose.walmClient.GetProject(namespace, compose.projectName)
		if err != nil {
			klog.Errorf("get project error %v", resp)
		} else {
			projectResp := project.ProjectInfo{}
			if resp.IsSuccess() {
				_ = json.Unmarshal(resp.Body(), projectResp)
				if projectResp.Ready {
					klog.Infof("project %s/%s is ready", namespace, compose.projectName)
					break
				}
			} else {
				klog.Errorf("project %s/%s error %s", namespace, compose.projectName, resp.Body())
			}
		}
		if time.Since(startTime) > timeout {
			klog.Infof("project %s/%s wait ready timeout...", namespace, compose.projectName)
			return errors.New(fmt.Sprintf("project %s/%s wait ready timeout", namespace, compose.projectName))
		} else {
			klog.Infof("project %s/%s not ready waiting...", namespace, compose.projectName)
			time.Sleep(5 * time.Second)
		}
	}

	return nil
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
