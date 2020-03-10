// Copyright 2018 The transwarp/monitor-crd-informer Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	time "time"

	monitoringv1 "transwarp/monitor-crd-informer/pkg/apis/monitoring/v1"
	internalinterfaces "transwarp/monitor-crd-informer/pkg/client/informers/externalversions/internalinterfaces"
	v1 "transwarp/monitor-crd-informer/pkg/client/listers/monitoring/v1"
	versioned "transwarp/monitor-crd-informer/pkg/client/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// ServiceMonitorInformer provides access to a shared informer and lister for
// ServiceMonitors.
type ServiceMonitorInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1.ServiceMonitorLister
}

type serviceMonitorInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewServiceMonitorInformer constructs a new informer for ServiceMonitor type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewServiceMonitorInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredServiceMonitorInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredServiceMonitorInformer constructs a new informer for ServiceMonitor type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredServiceMonitorInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MonitoringV1().ServiceMonitors(namespace).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MonitoringV1().ServiceMonitors(namespace).Watch(options)
			},
		},
		&monitoringv1.ServiceMonitor{},
		resyncPeriod,
		indexers,
	)
}

func (f *serviceMonitorInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredServiceMonitorInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *serviceMonitorInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&monitoringv1.ServiceMonitor{}, f.defaultInformer)
}

func (f *serviceMonitorInformer) Lister() v1.ServiceMonitorLister {
	return v1.NewServiceMonitorLister(f.Informer().GetIndexer())
}
