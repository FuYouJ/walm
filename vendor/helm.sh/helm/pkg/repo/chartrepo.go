/*
Copyright The Helm Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package repo // import "helm.sh/helm/pkg/repo"

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/yaml"

	"helm.sh/helm/pkg/chart/loader"
	"helm.sh/helm/pkg/getter"
	"helm.sh/helm/pkg/helmpath"
	"helm.sh/helm/pkg/provenance"
)

// Entry represents a collection of parameters for chart repository
type Entry struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
	CAFile   string `json:"caFile"`
}

// ChartRepository represents a chart repository
type ChartRepository struct {
	Config     *Entry
	ChartPaths []string
	IndexFile  *IndexFile
	Client     getter.Getter
}

// NewChartRepository constructs ChartRepository
func NewChartRepository(cfg *Entry, getters getter.Providers) (*ChartRepository, error) {
	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, errors.Errorf("invalid chart URL format: %s", cfg.URL)
	}

	getterConstructor, err := getters.ByScheme(u.Scheme)
	if err != nil {
		return nil, errors.Errorf("could not find protocol handler for: %s", u.Scheme)
	}
	client, err := getterConstructor(
		getter.WithURL(cfg.URL),
		getter.WithTLSClientConfig(cfg.CertFile, cfg.KeyFile, cfg.CAFile),
		getter.WithBasicAuth(cfg.Username, cfg.Password),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "could not construct protocol handler for: %s", u.Scheme)
	}

	return &ChartRepository{
		Config:    cfg,
		IndexFile: NewIndexFile(),
		Client:    client,
	}, nil
}

// Load loads a directory of charts as if it were a repository.
//
// It requires the presence of an index.yaml file in the directory.
func (r *ChartRepository) Load() error {
	dirInfo, err := os.Stat(r.Config.Name)
	if err != nil {
		return err
	}
	if !dirInfo.IsDir() {
		return errors.Errorf("%q is not a directory", r.Config.Name)
	}

	// FIXME: Why are we recursively walking directories?
	// FIXME: Why are we not reading the repositories.yaml to figure out
	// what repos to use?
	filepath.Walk(r.Config.Name, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			if strings.Contains(f.Name(), "-index.yaml") {
				i, err := LoadIndexFile(path)
				if err != nil {
					return nil
				}
				r.IndexFile = i
			} else if strings.HasSuffix(f.Name(), ".tgz") {
				r.ChartPaths = append(r.ChartPaths, path)
			}
		}
		return nil
	})
	return nil
}

// DownloadIndexFile fetches the index from a repository.
func (r *ChartRepository) DownloadIndexFile() error {
	var indexURL string
	parsedURL, err := url.Parse(r.Config.URL)
	if err != nil {
		return err
	}
	parsedURL.Path = strings.TrimSuffix(parsedURL.Path, "/") + "/index.yaml"

	indexURL = parsedURL.String()
	// TODO add user-agent
	resp, err := r.Client.Get(indexURL)
	if err != nil {
		return err
	}

	index, err := ioutil.ReadAll(resp)
	if err != nil {
		return err
	}

	if _, err := loadIndex(index); err != nil {
		return err
	}

	return ioutil.WriteFile(helmpath.CacheIndex(r.Config.Name), index, 0644)
}

// Index generates an index for the chart repository and writes an index.yaml file.
func (r *ChartRepository) Index() error {
	err := r.generateIndex()
	if err != nil {
		return err
	}
	return r.saveIndexFile()
}

func (r *ChartRepository) saveIndexFile() error {
	index, err := yaml.Marshal(r.IndexFile)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(r.Config.Name, indexPath), index, 0644)
}

func (r *ChartRepository) generateIndex() error {
	for _, path := range r.ChartPaths {
		ch, err := loader.Load(path)
		if err != nil {
			return err
		}

		digest, err := provenance.DigestFile(path)
		if err != nil {
			return err
		}

		if !r.IndexFile.Has(ch.Name(), ch.Metadata.Version) {
			r.IndexFile.Add(ch.Metadata, path, r.Config.URL, digest)
		}
		// TODO: If a chart exists, but has a different Digest, should we error?
	}
	r.IndexFile.SortEntries()
	return nil
}

// FindChartInRepoURL finds chart in chart repository pointed by repoURL
// without adding repo to repositories
func FindChartInRepoURL(repoURL, chartName, chartVersion, certFile, keyFile, caFile string, getters getter.Providers) (string, error) {
	return FindChartInAuthRepoURL(repoURL, "", "", chartName, chartVersion, certFile, keyFile, caFile, getters)
}

// FindChartInAuthRepoURL finds chart in chart repository pointed by repoURL
// without adding repo to repositories, like FindChartInRepoURL,
// but it also receives credentials for the chart repository.
func FindChartInAuthRepoURL(repoURL, username, password, chartName, chartVersion, certFile, keyFile, caFile string, getters getter.Providers) (string, error) {

	// Download and write the index file to a temporary location
	buf := make([]byte, 20)
	rand.Read(buf)
	name := strings.Replace(base64.StdEncoding.EncodeToString(buf), "/", "-", -1)

	c := Entry{
		URL:      repoURL,
		Username: username,
		Password: password,
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
		Name:     name,
	}
	r, err := NewChartRepository(&c, getters)
	if err != nil {
		return "", err
	}
	if err := r.DownloadIndexFile(); err != nil {
		return "", errors.Wrapf(err, "looks like %q is not a valid chart repository or cannot be reached", repoURL)
	}

	// Read the index file for the repository to get chart information and return chart URL
	repoIndex, err := LoadIndexFile(helmpath.CacheIndex(name))
	if err != nil {
		return "", err
	}

	errMsg := fmt.Sprintf("chart %q", chartName)
	if chartVersion != "" {
		errMsg = fmt.Sprintf("%s version %q", errMsg, chartVersion)
	}
	cv, err := repoIndex.Get(chartName, chartVersion)
	if err != nil {
		return "", errors.Errorf("%s not found in %s repository", errMsg, repoURL)
	}

	if len(cv.URLs) == 0 {
		return "", errors.Errorf("%s has no downloadable URLs", errMsg)
	}

	chartURL := cv.URLs[0]

	absoluteChartURL, err := ResolveReferenceURL(repoURL, chartURL)
	if err != nil {
		return "", errors.Wrap(err, "failed to make chart URL absolute")
	}

	return absoluteChartURL, nil
}

// ResolveReferenceURL resolves refURL relative to baseURL.
// If refURL is absolute, it simply returns refURL.
func ResolveReferenceURL(baseURL, refURL string) (string, error) {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse %s as URL", baseURL)
	}

	parsedRefURL, err := url.Parse(refURL)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse %s as URL", refURL)
	}

	// We need a trailing slash for ResolveReference to work, but make sure there isn't already one
	parsedBaseURL.Path = strings.TrimSuffix(parsedBaseURL.Path, "/") + "/"
	return parsedBaseURL.ResolveReference(parsedRefURL).String(), nil
}