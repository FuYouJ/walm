package k8s

import (
	"WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/models/release"
)

type Operator interface {
	DeletePod(namespace string, name string) error
	// diff between delete and restart: if pod does not exist, restart return err, but delete not
	RestartPod(namespace string, name string) error
	MigratePod(mig *k8s.Mig) error
	DeletePodMigration(namespace string, name string) error
	MigrateNode(srcNode string, destNode string) error

	BuildManifestObjects(namespace string, manifest string) ([]map[string]interface{}, error)
	ComputeReleaseResourcesByManifest(namespace string, manifest string) (*release.ReleaseResources, error)

	CreateNamespace(namespace *k8s.Namespace) error
	UpdateNamespace(namespace *k8s.Namespace) (error)
	DeleteNamespace(name string) error

	CreateResourceQuota(resourceQuota *k8s.ResourceQuota) error
	CreateOrUpdateResourceQuota(resourceQuota *k8s.ResourceQuota) error

	CreateLimitRange(limitRange *k8s.LimitRange) error

	LabelNode(name string, labelsToAdd map[string]string, labelsToRemove []string) (error)
	AnnotateNode(name string, annotationsToAdd map[string]string, annotationsToRemove []string) (error)
	TaintNoExecuteNode(name string, taintsToAdd map[string]string, taintsToRemove []string) (error)

	DeleteStatefulSetPvcs(statefulSets []*k8s.StatefulSet) error
	DeleteIsomateSetPvcs(isomateSets []*k8s.IsomateSet) error
	DeletePvc(namespace string, name string) error
	DeletePvcs(namespace string, labelSeletorStr string) error

	CreateSecret(namespace string, secretRequestBody *k8s.CreateSecretRequestBody) error
	UpdateSecret(namespace string, secretRequestBody *k8s.CreateSecretRequestBody) error
	DeleteSecret(namespace, name string) (err error)

	CreateService(namespace string, serviceRequestBody *k8s.CreateServiceRequestBody) error
	UpdateService(namespace string, serviceRequestBody *k8s.CreateServiceRequestBody, fullUpdate bool) error
	DeleteService(namespacae, name string) (err error)
	UpdateIngress(namespace, ingressName string, requestBody *k8s.IngressRequestBody) error
	UpdateConfigMap(namespace, configMapName string, requestBody *k8s.ConfigMapRequestBody) error
}
