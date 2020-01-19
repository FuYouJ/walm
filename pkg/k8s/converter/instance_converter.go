package converter

import (
	"WarpCloud/walm/pkg/models/k8s"
	"transwarp/application-instance/pkg/apis/transwarp/v1beta1"
	"WarpCloud/walm/pkg/release/utils"
)

func ConvertInstanceFromK8s(oriInst *v1beta1.ApplicationInstance, instModules *k8s.ResourceSet, dependencyMeta *k8s.DependencyMeta) (*k8s.ApplicationInstance, error) {
	if oriInst == nil {
		return nil, nil
	}
	inst := oriInst.DeepCopy()
	return &k8s.ApplicationInstance{
		Meta:              k8s.NewMeta(k8s.InstanceKind, inst.Namespace, inst.Name, k8s.NewState("Ready", "", "")),
		CreationTimestamp: inst.CreationTimestamp.String(),
		InstanceId:        inst.Spec.InstanceId,
		Dependencies:      convertInstDependencies(inst.Namespace, inst.Spec.Dependencies),
		DependencyMeta:    dependencyMeta,
		Modules:           instModules,
	}, nil
}

func convertInstDependencies(namespace string, dependencies []v1beta1.Dependency) map[string]string {
	res := map[string]string{}
	for _, dep := range dependencies {
		if dep.DependencyRef.Namespace == namespace {
			res[dep.Name] = dep.DependencyRef.Name
		} else {
			res[dep.Name] = dep.DependencyRef.Namespace + utils.ReleaseSep + dep.DependencyRef.Name
		}
	}
	return res
}
