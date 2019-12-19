package main

import (
	"WarpCloud/walm/cmd/walmctl/util/walmctlclient"
	"WarpCloud/walm/pkg/helm/impl"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/setting"
	"encoding/json"
	"fmt"
	"github.com/ghodss/yaml"
	"github.com/go-resty/resty"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tidwall/gjson"
	"golang.org/x/crypto/ssh"
	"helm.sh/helm/pkg/registry"
	"helm.sh/helm/pkg/repo"
	"io"
	"io/ioutil"
	"k8s.io/klog"
	"k8s.io/kubernetes/pkg/kubectl/util/i18n"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type syncCmd struct {
	name       string
	file       string
	clusterIP  string
	user       string
	password   string
	sshPort    string
	kubeconfig string
	out        io.Writer
}

func newSyncCmd(out io.Writer) *cobra.Command {
	sync := &syncCmd{out: out}
	cmd := &cobra.Command{
		Use:                   "sync release",
		DisableFlagsInUseLine: false,
		Short:                 i18n.T("sync instance of your app to other k8s cluster or save into files"),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.Errorf("Arguments invalid, format like `sync release zookeeper-test` instead")
			}
			if args[0] != "release" {
				return errors.Errorf("Unsupported sync type, release only currently")
			}
			if namespace == "" {
				return errNamespaceRequired
			}
			sync.name = args[1]
			return sync.run()
		},
	}
	cmd.PersistentFlags().StringVar(&sync.file, "save", "/tmp/walm-sync", "filepath to save instance of app")
	cmd.PersistentFlags().StringVar(&sync.clusterIP, "cluster-ip", "", "ip address for k8s cluster")
	cmd.PersistentFlags().StringVarP(&sync.user, "user", "u", "root", "user for k8s cluster")
	cmd.PersistentFlags().StringVarP(&sync.password, "password", "p", "", "password for k8s cluster")
	cmd.PersistentFlags().StringVar(&sync.sshPort, "sshPort", "22", "sshPort, default 22")
	return cmd
}

func (sync *syncCmd) run() error {
	client := walmctlclient.CreateNewClient(walmserver)
	if err := client.ValidateHostConnect(); err != nil {
		return err
	}

	resp, err := client.GetRelease(namespace, sync.name)
	if err != nil {
		return err
	}

	var releaseInfo release.ReleaseInfoV2
	err = json.Unmarshal(resp.Body(), &releaseInfo)
	if err != nil {
		return err
	}

	dirName := releaseInfo.Namespace + "-" + releaseInfo.Name
	targetDir := filepath.Join(sync.file, dirName)
	configValuesByte, err := json.Marshal(releaseInfo.ConfigValues)
	if err != nil {
		klog.Errorf("failed to write configValues.yaml : %s", err.Error())
		return err
	}
	tmpDir, err := createTempDir()
	if err != nil {
		return err
	}
	tmpCvPath := filepath.Join(tmpDir, "configValues.yaml")
	if err := ioutil.WriteFile(tmpCvPath, configValuesByte, 0644); err != nil {
		klog.Errorf("failed to write chart : %s", err.Error())
		return err
	}

	chartName, err := saveCharts(client, releaseInfo, tmpDir)
	if err != nil {
		return err
	}
	tmpChartPath := filepath.Join(tmpDir, chartName)

	// transfer files to target cluster
	if sync.clusterIP != "" {
		sshConfig := &ssh.ClientConfig{
			User: sync.user,
			Auth: []ssh.AuthMethod{
				ssh.Password(sync.password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		sshClient, err := ssh.Dial("tcp", sync.clusterIP+":"+sync.sshPort, sshConfig)
		if err != nil {
			klog.Errorf("%s sshPort %s not reachable: %s", sync.clusterIP, sync.sshPort, err.Error())
			return err
		}
		defer sshClient.Close()
		session, err := sshClient.NewSession()
		if err != nil {
			klog.Errorf("failed to create session: %s", err.Error())
			return err
		}
		defer session.Close()
		chartFile, _ := os.Open(tmpCvPath)
		defer chartFile.Close()
		chartStat, _ := chartFile.Stat()
		cvFile, _ := os.Open(tmpChartPath)
		defer cvFile.Close()
		cvStat, _ := cvFile.Stat()
		go func() {
			w, _ := session.StdinPipe()
			defer w.Close()
			_, err = fmt.Fprintln(w, "D0755", 0, dirName) // mkdir
			if err != nil {
				klog.Errorf("file %s exists: %s", targetDir, err.Error())
			}
			fmt.Fprintf(w, "C0664 %d %s\n", chartStat.Size(), chartName)
			io.Copy(w, chartFile)
			fmt.Fprint(w, "\x00")
			fmt.Fprintf(w, "C0664 %d %s\n", cvStat.Size(), "configValues.yaml")
			io.Copy(w, cvFile)
			fmt.Fprint(w, "\x00")
		}()
		if err := session.Run("/usr/bin/scp -tr " + targetDir); err != nil {
			return errors.Errorf("failed to run: %s  may not exist in %s", targetDir, sync.clusterIP)
		}
	} else {
		if _, err = copyFile(tmpCvPath, filepath.Join(targetDir, "configValues.yaml")); err != nil {
			return errors.Errorf("failed to copy configValues.yaml to %s: %s", targetDir, err.Error())
		}

		if _, err = copyFile(tmpChartPath, filepath.Join(targetDir, chartName)); err != nil {
			return errors.Errorf("failed to copy %s to %s: %s", chartName, targetDir, err.Error())
		}
	}
	err = os.RemoveAll(tmpDir)
	if err != nil {
		return errors.Errorf("failed to remove tmp dir: %s", err.Error())
	}
	return nil
}

func saveCharts(client *walmctlclient.WalmctlClient, releaseInfo release.ReleaseInfoV2, tmpDir string) (string, error) {

	chartImage := releaseInfo.ChartImage
	chartRepo := releaseInfo.RepoName
	chartName := releaseInfo.ChartName
	chartVersion := releaseInfo.ChartVersion
	name := ""
	registryClient, err := impl.NewRegistryClient(&setting.ChartImageConfig{CacheRootDir: "/chart-cache"})
	if err != nil {
		return "", err
	}
	if chartImage != "" {
		ref, err := registry.ParseReference(chartImage)
		if err != nil {
			klog.Errorf("failed to parse chart image %s : %s", chartImage, err.Error())
			return "", errors.Wrapf(err, "failed to parse chart image %s", chartImage)
		}
		err = registryClient.PullChart(ref)
		if err != nil {
			klog.Errorf("failed to push chart image : %s", err.Error())
			return "", err
		}
	} else {
		resp, err := client.GetRepoList()
		if err != nil {
			return "", err
		}

		var repoUrl string
		repos := gjson.Get(string(resp.Body()), "items").Array()
		for _, repo := range repos {
			if repo.Get("repoName").String() == chartRepo {
				repoUrl = repo.Get("repoUrl").String()
				break
			}
		}

		repoIndex := &repo.IndexFile{}
		chartInfoList := new(release.ChartInfoList)
		chartInfoList.Items = make([]*release.ChartInfo, 0)
		parsedURL, err := url.Parse(repoUrl)
		if err != nil {
			return "", err
		}
		parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/") + "/index.yaml"

		indexURL := parsedURL.String()

		resp, err = resty.R().Get(indexURL)
		if err != nil {
			klog.Errorf("failed to get index : %s", err.Error())
			return "", err
		}

		if err := yaml.Unmarshal(resp.Body(), repoIndex); err != nil {
			return "", err
		}
		cv, err := repoIndex.Get(chartName, chartVersion)
		if err != nil {
			return "", fmt.Errorf("chart %s-%s is not found: %s", chartName, chartVersion, err.Error())
		}
		if len(cv.URLs) == 0 {
			return "", fmt.Errorf("chart %s has no downloadable URLs", chartName)
		}
		chartUrl := cv.URLs[0]
		absoluteChartURL, err := repo.ResolveReferenceURL(repoUrl, chartUrl)
		if err != nil {
			return "", fmt.Errorf("failed to make absolute chart url: %v", err)
		}
		resp, err = resty.R().Get(absoluteChartURL)
		if err != nil {
			klog.Errorf("failed to get chart : %s", err.Error())
			return "", err
		}
		name = filepath.Base(absoluteChartURL)
		if err := ioutil.WriteFile(filepath.Join(tmpDir, name), resp.Body(), 0644); err != nil {
			klog.Errorf("failed to write chart : %s", err.Error())
			return "", err
		}
	}

	return name, nil
}
