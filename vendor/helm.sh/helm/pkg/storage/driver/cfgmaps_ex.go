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

package driver

import (
	"fmt"
	rspb "helm.sh/helm/pkg/release"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kblabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"strconv"
	"strings"
	"time"
)

var _Driver = (*ConfigMapsEx)(nil)

// ConfigMapsExDriverName is the string name of the drive
const (
	ConfigMapsExDriverName = "ConfigMapEx"
	KubeSystemNamespace    = "kube-system"
)

// ConfigMapEx is a wrapper around an implementation of a kubernetes
// ConfigMapsInterface
type ConfigMapsEx struct {
	// release ns client
	impl corev1.ConfigMapInterface

	// kube-system ns client
	compatibleImpl corev1.ConfigMapInterface
	Log            func(string, ...interface{})
	// release ns
	ns string
}

func (cfgmaps *ConfigMapsEx) Create (key string, rls *rspb.Release) error {
	// set labels for configmaps object meta data
	var lbs labels

	lbs.init()
	lbs.set("CREATED_AT", strconv.Itoa(int(time.Now().Unix())))

	// create a new configmap to hold the release
	obj, err := newConfigMapsObject(key, rls, lbs)
	if err != nil {
		cfgmaps.Log("create: failed to encode release %q: %s", rls.Name, err)
		return err
	}
	// push the configmap object out into the kubiverse
	if _, err := cfgmaps.impl.Create(obj); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return ErrReleaseExists
		}
		cfgmaps.Log("create: failed to create: %s", err)
		return err
	}
	return nil
}

func (cfgmaps *ConfigMapsEx) Update(key string, rls *rspb.Release) error {
	// set labels for configmaps object meta data
	var lbs labels
	lbs.init()
	lbs.set("MODIFIED_AT", strconv.Itoa(int(time.Now().Unix())))

	// create a new configmap object to hold the release
	obj, err := newConfigMapsObject(key, rls, lbs)
	if err != nil {
		cfgmaps.Log("update: failed to encode release %q: %s", rls.Name, err)
		return err
	}

	// push the configmap object out into the kubiverse
	_, err = cfgmaps.impl.Update(obj)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if cfgmaps.ns != KubeSystemNamespace {
				_, err = cfgmaps.compatibleImpl.Update(obj)
				if err != nil {
					cfgmaps.Log("update: failed to update: %s", err)
					return err
				}
			} else {
				cfgmaps.Log("update: failed to update: %s", err)
				return err
			}
		} else {
			cfgmaps.Log("update: failed to update: %s", err)
			return err
		}
	}
	return nil

}

// Delete deletes the ConfigMap holding the release named by key
func (cfgmaps *ConfigMapsEx) Delete(key string) (rls *rspb.Release, err error) {
	// fetch the release to check existence
	if rls, err = cfgmaps.Get(key); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrReleaseExists
		}

		cfgmaps.Log("delete: failed to get release %q: %s", key, err)
		return nil, err
	}
	// delete the release

	if err = cfgmaps.impl.Delete(key, &metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			if cfgmaps.ns != KubeSystemNamespace {
				err = cfgmaps.compatibleImpl.Delete(key, &metav1.DeleteOptions{})
				if err != nil {
					return rls, err
				}
			} else {
				return rls, err
			}
		}
		return rls, err
	}
	return rls, nil
}

func (cfgmaps *ConfigMapsEx) Query(labels map[string]string) ([]*rspb.Release, error) {
	ls := kblabels.Set{}
	for k, v := range labels {
		if errs := validation.IsValidLabelValue(v); len(errs) != 0 {
			return nil, fmt.Errorf("invalid label value: %q: %s", v, strings.Join(errs, "; "))
		}
		ls[k] = v
	}
	opts := metav1.ListOptions{LabelSelector: ls.AsSelector().String()}

	list, err := cfgmaps.impl.List(opts)
	if err != nil {
		cfgmaps.Log("query: failed to query with labels: %s", err)
		return nil, err
	}

	var items []v1.ConfigMap
	if len(list.Items) > 0 {
		items = append(items, list.Items...)
	}

	if cfgmaps.ns != KubeSystemNamespace {
		list, err = cfgmaps.compatibleImpl.List(opts)
		if err != nil {
			cfgmaps.Log("list: failed to list: %s", err)
			return nil, err
		}
		if len(list.Items) > 0 {
			items = append(items, list.Items...)
		}
	}

	if len(items) == 0 {
		return nil, ErrReleaseNotFound
	}

	var results []*rspb.Release
	for _, item := range items {
		rls, err := decodeReleaseCompatible(item.Data["release"], item.Labels)
		if err != nil {
			cfgmaps.Log("query: failed to decode release: %s", err)
			continue
		}
		if rls.Namespace == cfgmaps.ns {
			results = append(results, rls)
		}
	}

	if len(results) == 0 {
		return nil, ErrReleaseNotFound
	}

	return results, nil
}

// NewConfigMapsEx initializes a new ConfigMapsEx wrapping an implementation of
// the kubernetes ConfigMapsInterface.
func NewConfigMapsEx(impl corev1.ConfigMapInterface, compatibleImpl corev1.ConfigMapInterface, ns string) *ConfigMapsEx {
	return &ConfigMapsEx{
		impl:           impl,
		compatibleImpl: compatibleImpl,
		Log:            func(_ string, _ ...interface{}) {},
		ns:             ns,
	}
}

// Name returns the name of the driver.
func (cfgmaps *ConfigMapsEx) Name() string {
	return ConfigMapsExDriverName
}

// Get fetches the release named by key. The corresponding release is returned
// or error if not found.
func (cfgmaps *ConfigMapsEx) Get(key string) (*rspb.Release, error) {
	// fetch the configmap holding the release named by key
	obj, err := cfgmaps.impl.Get(key, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			if cfgmaps.ns == KubeSystemNamespace {
				return nil, ErrReleaseNotFound
			} else {
				obj, err = cfgmaps.compatibleImpl.Get(key, metav1.GetOptions{})
				if err != nil {
					if apierrors.IsNotFound(err) {
						return nil, ErrReleaseNotFound
					}

					cfgmaps.Log("get: failed to get %q: %s", key, err)
					return nil, err
				}
			}
		} else {
			cfgmaps.Log("get: failed to get %q: %s", key, err)
			return nil, err
		}
	}
	// found the configmap, decode the based64 data string
	r, err := decodeReleaseCompatible(obj.Data["release"], obj.Labels)
	if err != nil {
		cfgmaps.Log("get: failed to decode data %q: %s", key, err)
		return nil, err
	}
	if r.Namespace != cfgmaps.ns {
		return nil, ErrReleaseNotFound
	}
	// return the release object
	return r, nil
}

// List fetches all releases and returns the list releases such
// that filter(release) == true. An error is returned if the
// configmap fails to retrieve the releases.
func (cfgmaps *ConfigMapsEx) List(filter func(*rspb.Release) bool) ([]*rspb.Release, error) {

	lselArray := []kblabels.Set{
		{
			"owner": "helm",
		},
		{
			"OWNER": "TILLER",
		},
		//Todo:// support upgraded v3 release
		//{
		//	"heritage": "Tiller",
		//},
	}
	var results []*rspb.Release
	for _, lsel := range lselArray {
		opts := metav1.ListOptions{LabelSelector: lsel.String()}

		list, err := cfgmaps.impl.List(opts)
		if err != nil {
			cfgmaps.Log("list: failed to list: %s", err)
			return nil, err
		}

		var items []v1.ConfigMap
		if len(list.Items) > 0 {
			items = append(items, list.Items...)
		}

		if cfgmaps.ns != "" && cfgmaps.ns != KubeSystemNamespace {
			list, err = cfgmaps.compatibleImpl.List(opts)
			if err != nil {
				cfgmaps.Log("list: failed to list: %s", err)
				return nil, err
			}
			if len(list.Items) > 0 {
				items = append(items, list.Items...)
			}
		}

		// iterate over the configmaps object list
		// and decode each release
		for _, item := range items {
			rls, err := decodeReleaseCompatible(item.Data["release"], lsel)
			if err != nil {
				cfgmaps.Log("list: failed to decode helm release: %v: %s", item, err)
				continue
			}

			if cfgmaps.ns != "" && rls.Namespace != cfgmaps.ns {
				continue
			}
			if filter(rls) {
				results = append(results, rls)
			}
		}
	}
	return results, nil
}
