package walmctlclient

import (
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/util"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty"
	"github.com/pkg/errors"
	"k8s.io/klog"
	"net"
	"strconv"
	"time"
)

type WalmctlClient struct {
	protocol   string
	hostURL    string
	apiVersion string
	baseURL    string
}

var walmctlClient *WalmctlClient

func CreateNewClient(hostURL string) *WalmctlClient {
	if walmctlClient == nil {
		walmctlClient = &WalmctlClient{
			protocol:   "http://",
			hostURL:    hostURL,
			apiVersion: "/api/v1",
		}
		walmctlClient.baseURL = walmctlClient.protocol + walmctlClient.hostURL + walmctlClient.apiVersion
	}
	return walmctlClient
}

func (c *WalmctlClient) ValidateHostConnect() error {
	timeout := time.Duration(5 * time.Second)
	_, err := net.DialTimeout("tcp", walmctlClient.hostURL, timeout)
	if err != nil {
		return errors.Errorf("WalmServer unreachable, error: %s", err.Error())
	}
	return nil
}

func (c *WalmctlClient) CreateTenantIfNotExist(namespace string) error {
	fullUrl := walmctlClient.baseURL + "/tenant/" + namespace

	_, _ = resty.R().
		SetHeader("Content-Type", "application/json").
		SetBody("{}").
		Post(fullUrl)

	resp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		Get(fullUrl)

	if err != nil || !resp.IsSuccess() {
		return errors.Errorf("create Tenant Error %v", err)
	}
	return nil
}

func (c *WalmctlClient) CreateSecret(namespace, secretName string, secretData map[string]string) error {
	_ = c.CreateTenantIfNotExist(namespace)
	secretFullUrl := walmctlClient.baseURL + "/secret/" + namespace

	secretReq := k8sModel.CreateSecretRequestBody{
		Data: secretData,
		Type: "Opaque",
		Name: secretName,
	}
	resp, err := resty.R().SetHeader("Content-Type", "application/json").
		SetBody(secretReq).
		Post(secretFullUrl)
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		return errors.New(resp.String())
	}

	return nil
}

func (c *WalmctlClient) DeleteSecret(namespace, secretName string) error {
	_ = c.CreateTenantIfNotExist(namespace)
	secretFullUrl := walmctlClient.baseURL + "/secret/" + namespace + "/name/" + secretName

	resp, err := resty.R().SetHeader("Content-Type", "application/json").
		Delete(secretFullUrl)
	if err != nil {
		return err
	}
	if resp.StatusCode() != 200 {
		return errors.New(resp.String())
	}

	return nil
}

func (c *WalmctlClient) DryRunCreateRelease(
	namespace, chart string, releaseName string,
	configValues map[string]interface{},
) (*resty.Response, error) {
	fullUrl := walmctlClient.baseURL + "/release/" + namespace + "/dryrun"

	if releaseName != "" {
		releaseNameConfigs := make(map[string]interface{}, 0)
		releaseNameConfigs["name"] = releaseName
		util.MergeValues(configValues, releaseNameConfigs, false)
	}
	filestr, err := json.Marshal(configValues)
	if err != nil {
		klog.Errorf("marshal to json error %v", err)
	}

	resp := &resty.Response{}
	if chart != "" {
		chartFullUrl := walmctlClient.baseURL + "/release/" + namespace + "/dryrun/withchart"
		resp, err = resty.R().
			SetHeader("Content-Type", "multipart/form-data").
			SetFile("chart", chart).
			SetFormData(map[string]string{
				"namespace": namespace,
				"release":   releaseName,
				"body":      string(filestr[:]),
			}).
			Post(chartFullUrl)
	} else {
		resp, err = resty.R().
			SetHeader("Content-Type", "application/json").
			SetBody(filestr).
			Post(fullUrl)
	}
	if resp == nil || resp.StatusCode() != 200 {
		return nil, errors.New(fmt.Sprintf("error response %v %v", err, resp))
	}
	return resp, err
}

// release
func (c *WalmctlClient) CreateRelease(
	namespace, chart string, releaseName string,
	async bool, timeoutSec int64,
	configValues map[string]interface{},
) (*resty.Response, error) {
	_ = c.CreateTenantIfNotExist(namespace)
	fullUrl := walmctlClient.baseURL + "/release/" + namespace + "?async=" + strconv.FormatBool(async) +
		"&timeoutSec=" + strconv.FormatInt(timeoutSec, 10)

	if releaseName != "" {
		releaseNameConfigs := make(map[string]interface{}, 0)
		releaseNameConfigs["name"] = releaseName
		util.MergeValues(configValues, releaseNameConfigs, false)
	}
	filestr, err := json.Marshal(configValues)
	if err != nil {
		klog.Errorf("marshal to json error %v", err)
	}

	resp := &resty.Response{}
	if chart != "" {
		chartFullUrl := walmctlClient.baseURL + "/release/" + namespace + "/withchart"
		resp, err = resty.R().
			SetHeader("Content-Type", "multipart/form-data").
			SetFile("chart", chart).
			SetFormData(map[string]string{
				"release": releaseName,
				"body":    string(filestr[:]),
			}).
			Post(chartFullUrl)
	} else {
		resp, err = resty.R().
			SetHeader("Content-Type", "application/json").
			SetBody(filestr).
			Post(fullUrl)
	}
	if resp == nil || resp.StatusCode() != 200 {
		return nil, errors.New(fmt.Sprintf("error response %v %v", err, resp))
	}
	return resp, err
}

func (c *WalmctlClient) GetRelease(namespace string, releaseName string) (resp *resty.Response, err error) {
	fullUrl := walmctlClient.baseURL + "/release/" + namespace + "/name/" + releaseName

	resp, err = resty.R().Get(fullUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New(resp.String())
	}

	return resp, err
}

func (c *WalmctlClient) UpdateRelease(namespace string, newConfigStr string, async bool, timeoutSec int64) (resp *resty.Response, err error) {
	fullUrl := walmctlClient.baseURL + "/release/" + namespace + "?async=" + strconv.FormatBool(async) +
		"&timeoutSec=" + strconv.FormatInt(timeoutSec, 10)

	resp, err = resty.R().SetHeader("Content-Type", "application/json").
		SetBody(newConfigStr).
		Put(fullUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New(resp.String())
	}
	return resp, err
}

func (c *WalmctlClient) UpdateReleaseWithChart(namespace string, releaseName string, file string, newConfigStr string) (resp *resty.Response, err error) {
	fullUrl := walmctlClient.baseURL + "/release/" + namespace + "/withchart"

	resp, err = resty.R().SetHeader("Content-Type", "multipart/form-data", ).
		SetFile("chart", file).
		SetFormData(map[string]string{"release": releaseName, "body": newConfigStr}).
		Put(fullUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New(resp.String())
	}
	return resp, err
}

func (c *WalmctlClient) DeleteRelease(namespace string, releaseName string, async bool, timeoutSec int64, deletePvcs bool) (resp *resty.Response, err error) {
	fullUrl := walmctlClient.baseURL + "/release/" + namespace + "/name/" + releaseName + "?async=" + strconv.FormatBool(async) +
		"&timeoutSec=" + strconv.FormatInt(timeoutSec, 10) + "&deletePvcs=" + strconv.FormatBool(deletePvcs)

	resp, err = resty.R().
		Delete(fullUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New(resp.String())
	}
	return resp, err

}

func (c *WalmctlClient) ListRelease(namespace string, labelSelector string) (resp *resty.Response, err error) {
	fullUrl := walmctlClient.baseURL + "/release/" + namespace
	if namespace == "" {
		fullUrl = walmctlClient.baseURL + "/release"
	}

	resp, err = resty.R().
		SetHeader("Accept", "application/json").
		Get(fullUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New(resp.String())
	}
	return resp, err
}

// project
func (c *WalmctlClient) CreateProject(
	namespace, chartPath, projectName string,
	async bool, timeoutSec int64,
	configValues map[string]interface{},
) (resp *resty.Response, err error) {
	_ = c.CreateTenantIfNotExist(namespace)
	fullUrl := walmctlClient.baseURL + "/project/" + namespace + "/name/" + projectName + "?async=" + strconv.FormatBool(async) +
		"&timeoutSec=" + strconv.FormatInt(timeoutSec, 10)

	filestr, err := json.Marshal(configValues)
	if err != nil {
		klog.Errorf("marshal to json error %v", err)
	}

	resp, err = resty.R().SetHeader("Content-Type", "application/json").
		SetBody(filestr).
		Post(fullUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New(resp.String())
	}

	return resp, err
}

func (c *WalmctlClient) GetProject(namespace string, projectName string) (resp *resty.Response, err error) {
	fullUrl := walmctlClient.baseURL + "/project/" + namespace + "/name/" + projectName
	resp, err = resty.R().Get(fullUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New(resp.String())
	}
	return resp, err
}

func (c *WalmctlClient) DeleteProject(namespace string, projectName string, async bool, timeoutSec int64, deletePvcs bool) (resp *resty.Response, err error) {
	fullUrl := walmctlClient.baseURL + "/project/" + namespace + "/name/" + projectName + "?async=" + strconv.FormatBool(async) +
		"&timeoutSec=" + strconv.FormatInt(timeoutSec, 10) + "&deletePvcs=" + strconv.FormatBool(deletePvcs)

	resp, err = resty.R().
		Delete(fullUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New(resp.String())
	}

	return resp, err
}

func (c *WalmctlClient) ListProject(namespace string) (resp *resty.Response, err error) {
	fullUrl := walmctlClient.baseURL + "/project/" + namespace
	if namespace == "" {
		fullUrl = walmctlClient.baseURL + "/project"
	}

	resp, err = resty.R().
		SetHeader("Accept", "application/json").
		Get(fullUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New(resp.String())
	}

	return resp, err
}

func (c *WalmctlClient) AddReleaseInProject(namespace string, releaseName string, projectName string, async bool, timeoutSec int64, configValues map[string]interface{}) (resp *resty.Response, err error) {
	if releaseName != "" {
		releaseNameConfigs := make(map[string]interface{}, 0)
		releaseNameConfigs["name"] = releaseName
		util.MergeValues(configValues, releaseNameConfigs, false)
	}
	fileStr, err := json.Marshal(configValues)
	if err != nil {
		klog.Errorf("marshal to json error %v", err)
	}

	fullUrl := walmctlClient.baseURL + "/project/" + namespace + "/name/" + projectName + "/instance?async=" + strconv.FormatBool(async) + "&timeoutSec=" + strconv.FormatInt(timeoutSec, 10)
	resp, err = resty.R().SetHeader("Content-Type", "application/json").
		SetBody(fileStr).
		Post(fullUrl)

	return resp, err
}

func (c *WalmctlClient) DeleteReleaseInProject(namespace string, projectName string, releaseName string, async bool, timeoutSec int64, deletePvcs bool) (resp *resty.Response, err error) {
	fullUrl := walmctlClient.baseURL + "/project/" + namespace + "/name/" + projectName + "/instance/" + releaseName + "?async=" + strconv.FormatBool(async) +
		"&timeoutSec=" + strconv.FormatInt(timeoutSec, 10) + "&deletePvcs=" + strconv.FormatBool(deletePvcs)

	resp, err = resty.R().
		Delete(fullUrl)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != 200 {
		return nil, errors.New(resp.String())
	}

	return resp, err
}
