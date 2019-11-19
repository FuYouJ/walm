package k8s

import (
	"WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
	"WarpCloud/walm/pkg/models/tenant"
)

type Cache interface {
	GetResourceSet(releaseResourceMetas []release.ReleaseResourceMeta) (resourceSet *k8s.ResourceSet,err error)
	GetResource(kind k8s.ResourceKind, namespace, name string) (k8s.Resource, error)
	GetNodeMigration(node string) (*k8s.MigList, error)
	AddReleaseConfigHandler(OnAdd func(obj interface{}), OnUpdate func(oldObj, newObj interface{}), OnDelete func(obj interface{}))
	AddServiceHandler(OnAdd func(obj interface{}), OnUpdate func(oldObj, newObj interface{}), OnDelete func(obj interface{}))
	ListReleaseConfigs(namespace, labelSelectorStr string) ([]*k8s.ReleaseConfig, error)

	ListPersistentVolumeClaims(namespace string, labelSelectorStr string) ([]*k8s.PersistentVolumeClaim, error)

	ListTenants(labelSelectorStr string) (*tenant.TenantInfoList, error)
	GetTenant(tenantName string) (*tenant.TenantInfo, error)

	GetNodes(labelSelector string) ([]*k8s.Node, error)

	ListStatefulSets(namespace string, labelSelectorStr string) ([]*k8s.StatefulSet, error)

	GetPodEventList(namespace string, name string) (*k8s.EventList, error)
    GetDeploymentEventList(namespace string, name string) (*k8s.EventList, error)
	GetStatefulSetEventList(namespace string, name string) (*k8s.EventList, error)
	//GetReleaseEventList(namespace string, name string) (*k8s.EventList, error)
	GetPodLogs(namespace string, podName string, containerName string, tailLines int64) (string, error)

	ListSecrets(namespace string, name string) (*k8s.SecretList, error)

	ListStorageClasses(namespace string, labelSelectorStr string) ([]*k8s.StorageClass, error)
	ListServices(namespace string, labelSelectorStr string) ([]*k8s.Service, error)

	ListMigrations(labelSelectorStr string) (*k8s.MigList, error)
}
