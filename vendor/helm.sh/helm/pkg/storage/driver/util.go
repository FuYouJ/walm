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

package driver // import "helm.sh/helm/pkg/storage/driver"

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/golang/protobuf/ptypes/timestamp"
	"helm.sh/helm/pkg/chart"
	v2chrtutil "helm.sh/helm/pkg/chartutil"
	v2chart "helm.sh/helm/pkg/proto/hapi/chart"
	v2rls "helm.sh/helm/pkg/proto/hapi/release"
	rspb "helm.sh/helm/pkg/release"
	"io/ioutil"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"strings"
	"time"
)

var b64 = base64.StdEncoding

var magicGzip = []byte{0x1f, 0x8b, 0x08}

// encodeRelease encodes a release returning a base64 encoded
// gzipped binary protobuf encoding representation, or error.
func encodeRelease(rls *rspb.Release) (string, error) {
	b, err := json.Marshal(rls)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err = w.Write(b); err != nil {
		return "", err
	}
	w.Close()

	return b64.EncodeToString(buf.Bytes()), nil
}

// decodeRelease decodes the bytes in data into a release
// type. Data must contain a base64 encoded string of a
// valid protobuf encoding of a release, otherwise
// an error is returned.
// Todo:// Depracted(old)
func decodeRelease(data string) (*rspb.Release, error) {
	// base64 decode string
	b, err := b64.DecodeString(data)
	if err != nil {
		return nil, err
	}

	// For backwards compatibility with releases that were stored before
	// compression was introduced we skip decompression if the
	// gzip magic header is not found
	if bytes.Equal(b[0:3], magicGzip) {
		r, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		b2, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		b = b2
	}

	var rls rspb.Release
	// unmarshal protobuf bytes
	if err := json.Unmarshal(b, &rls); err != nil {
		return nil, err
	}
	return &rls, nil
}

func decodeReleaseCompatible(data string, lsel kblabels.Set) (*rspb.Release, error) {
	// base64 decode string
	b, err := b64.DecodeString(data)
	if err != nil {
		return nil, err
	}

	// For backwards compatibility with releases that were stored before
	// compression was introduced we skip decompression if the
	// gzip magic header is not found
	if bytes.Equal(b[0:3], magicGzip) {
		r, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		b2, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		b = b2
	}

	var rls rspb.Release
	var rlsV2 v2rls.Release

	// unmarshal protobuf bytes
	if lsel.Has("owner") && lsel.Get("owner") == "helm" {
		if err := json.Unmarshal(b, &rls); err != nil {
			return nil, err
		}
		rls.HelmVersion = "v3"
		return &rls, nil
	} else if lsel.Has("OWNER") && lsel.Get("OWNER") == "TILLER" {
		if err := proto.Unmarshal(b, &rlsV2); err != nil {
			return nil, err
		}
		return ConvertV2Release(&rlsV2, "v2")
	} else {
		// Todo
	}
	return nil, nil
}

// CreateRelease create a v3 release object from v3 release object
func ConvertV2Release(v2Rel *v2rls.Release, helmVersion string) (*rspb.Release, error) {
	if v2Rel.Chart == nil || v2Rel.Info == nil {
		return nil, fmt.Errorf("No v2 chart or info metadata")
	}
	chrt, err := mapv2ChartTov3Chart(v2Rel.Chart)
	if err != nil {
		return nil, err
	}
	config, err := mapConfig(v2Rel.Config)
	if err != nil {
		return nil, err
	}
	first, err := mapTimestampToTime(v2Rel.Info.FirstDeployed)
	if err != nil {
		return nil, err
	}
	last, err := mapTimestampToTime(v2Rel.Info.LastDeployed)
	if err != nil {
		return nil, err
	}
	deleted, err := mapTimestampToTime(v2Rel.Info.Deleted)
	if err != nil {
		return nil, err
	}
	status, ok := v2rls.Status_Code_name[int32(v2Rel.Info.Status.Code)]
	if !ok {
		return nil, fmt.Errorf("Failed to convert status")
	}
	hooks, err := mapHooks(v2Rel.Hooks)
	if err != nil {
		return nil, err
	}

	if v2Rel.GetHelmVersion() == "" {
		v2Rel.HelmVersion = helmVersion
	}

	return &rspb.Release{
		Name:      v2Rel.Name,
		Namespace: v2Rel.Namespace,
		Chart:     chrt,
		Config:    config,
		Info: &rspb.Info{
			FirstDeployed: first,
			LastDeployed:  last,
			Description:   v2Rel.Info.Description,
			Deleted:       deleted,
			Status:        rspb.Status(strings.ToLower(status)),
			Notes:         v2Rel.Info.Status.Notes,
		},
		Manifest:    v2Rel.Manifest,
		Hooks:       hooks,
		Version:     int(v2Rel.Version),
		HelmVersion: v2Rel.GetHelmVersion(),
	}, nil
}

// StoreRelease stores a release object in Helm v3 storage
//func StoreRelease(rel *release.Release) error {
//	cfg, err := GetActionConfig(rel.Namespace)
//	if err != nil {
//		return err
//	}
//
//	return cfg.Releases.Create(rel)
//}

func mapv2ChartTov3Chart(v2Chrt *v2chart.Chart) (*chart.Chart, error) {
	v3Chrt := new(chart.Chart)
	v3Chrt.Metadata = mapMetadata(v2Chrt)
	v3Chrt.Templates = mapTemplates(v2Chrt.Templates)
	err := mapDependencies(v2Chrt.Dependencies, v3Chrt)
	if err != nil {
		return nil, err
	}
	if v3Chrt.Values, err = mapConfig(v2Chrt.Values); err != nil {
		return nil, err
	}
	v3Chrt.Files = mapFiles(v2Chrt.Files)
	//TODO
	//v3Chrt.Schema
	//TODO
	//v3Chrt.Lock = new(chart.Lock)
	return v3Chrt, nil
}

func mapMetadata(v2Chrt *v2chart.Chart) *chart.Metadata {
	if v2Chrt.Metadata == nil {
		return nil
	}
	metadata := new(chart.Metadata)
	metadata.Name = v2Chrt.Metadata.Name
	metadata.Home = v2Chrt.Metadata.Home
	metadata.Sources = v2Chrt.Metadata.Sources
	metadata.Version = v2Chrt.Metadata.Version
	metadata.Description = v2Chrt.Metadata.Description
	metadata.Keywords = v2Chrt.Metadata.Keywords
	metadata.Maintainers = mapMaintainers(v2Chrt.Metadata.Maintainers)
	metadata.Icon = v2Chrt.Metadata.Icon
	metadata.APIVersion = v2Chrt.Metadata.ApiVersion
	metadata.Condition = v2Chrt.Metadata.Condition
	metadata.Tags = v2Chrt.Metadata.Tags
	metadata.AppVersion = v2Chrt.Metadata.AppVersion
	metadata.Deprecated = v2Chrt.Metadata.Deprecated
	metadata.Annotations = v2Chrt.Metadata.Annotations
	metadata.KubeVersion = v2Chrt.Metadata.KubeVersion
	//TODO: metadata.Dependencies =
	//Default to application
	metadata.Type = "application"
	return metadata
}

func mapMaintainers(v2Maintainers []*v2chart.Maintainer) []*chart.Maintainer {
	if v2Maintainers == nil {
		return nil
	}
	maintainers := []*chart.Maintainer{}
	for _, val := range v2Maintainers {
		maintainer := new(chart.Maintainer)
		maintainer.Name = val.Name
		maintainer.Email = val.Email
		maintainer.URL = val.Url
		maintainers = append(maintainers, maintainer)
	}
	return maintainers
}

func mapTemplates(v2Templates []*v2chart.Template) []*chart.File {
	if v2Templates == nil {
		return nil
	}
	files := []*chart.File{}
	for _, val := range v2Templates {
		file := new(chart.File)
		file.Name = val.Name
		file.Data = val.Data
		files = append(files, file)
	}
	return files
}

func mapDependencies(v2Dependencies []*v2chart.Chart, chart *chart.Chart) error {
	if v2Dependencies == nil {
		return nil
	}
	for _, val := range v2Dependencies {
		dependency, err := mapv2ChartTov3Chart(val)
		if err != nil {
			return err
		}
		chart.AddDependency(dependency)
	}
	return nil
}

func mapConfig(v2Config *v2chart.Config) (map[string]interface{}, error) {
	if v2Config == nil {
		return nil, nil
	}
	values, err := v2chrtutil.ReadValues([]byte(v2Config.Raw))
	if err != nil {
		return nil, err
	}
	return values, nil
}

func mapFiles(v2Files []*any.Any) []*chart.File {
	if mapFiles == nil {
		return nil
	}
	files := []*chart.File{}
	for _, f := range v2Files {
		file := new(chart.File)
		file.Name = f.TypeUrl
		file.Data = f.Value
		files = append(files, file)
	}
	return files
}

func mapHooks(v2Hooks []*v2rls.Hook) ([]*rspb.Hook, error) {
	if v2Hooks == nil {
		return nil, nil
	}
	hooks := []*rspb.Hook{}
	for _, val := range v2Hooks {
		hook := new(rspb.Hook)
		hook.Name = val.Name
		hook.Kind = val.Kind
		hook.Path = val.Path
		hook.Manifest = val.Manifest
		events, err := mapHookEvents(val.Events)
		if err != nil {
			return nil, err
		}
		hook.Events = events
		hook.Weight = int(val.Weight)
		if err != nil {
			return nil, err
		}
		policies, err := mapHookDeletePolicies(val.DeletePolicies)
		if err != nil {
			return nil, err
		}
		hook.DeletePolicies = policies
		//TODO: hook.LastRun =
		hooks = append(hooks, hook)
	}
	return hooks, nil
}

func mapHookEvents(v2HookEvents []v2rls.Hook_Event) ([]rspb.HookEvent, error) {
	if v2HookEvents == nil {
		return nil, nil
	}
	hookEvents := []rspb.HookEvent{}
	for _, val := range v2HookEvents {
		v2EventStr, ok := v2rls.Hook_Event_name[int32(val)]
		if !ok {
			return nil, fmt.Errorf("Failed to convert hook event")
		}
		event := rspb.HookEvent(strings.ToLower(v2EventStr))
		hookEvents = append(hookEvents, event)
	}
	return hookEvents, nil
}

func mapHookDeletePolicies(v2HookDelPolicies []v2rls.Hook_DeletePolicy) ([]rspb.HookDeletePolicy, error) {
	if v2HookDelPolicies == nil {
		return nil, nil
	}
	hookDelPolicies := []rspb.HookDeletePolicy{}
	for _, val := range v2HookDelPolicies {
		v2PolicyStr, ok := v2rls.Hook_DeletePolicy_name[int32(val)]
		if !ok {
			return nil, fmt.Errorf("Failed to convert hook delete policy")
		}
		policy := rspb.HookDeletePolicy(strings.ToLower(v2PolicyStr))
		hookDelPolicies = append(hookDelPolicies, policy)
	}
	return hookDelPolicies, nil
}

func mapTimestampToTime(ts *timestamp.Timestamp) (time.Time, error) {
	var mappedTime time.Time
	var err error
	if ts != nil {
		mappedTime, err = ptypes.Timestamp(ts)
		if err != nil {
			return mappedTime, err
		}
	}
	return mappedTime, nil
}
