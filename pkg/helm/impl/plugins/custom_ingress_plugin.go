package plugins

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog"
	"regexp"
	"strings"
)

const (
	CustomIngressPluginName = "CustomIngress"
)

const qnameCharFmt string = "[a-z0-9]"
const qnameExtCharFmt string = "[-a-z0-9_.]"
const qualifiedNameFmt string = "(" + qnameCharFmt + qnameExtCharFmt + "*)?" + qnameCharFmt
const qualifiedNameErrMsg string = "must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character"
const qualifiedNameMaxLength int = 63

var qualifiedNameRegexp = regexp.MustCompile("^" + qualifiedNameFmt + "$")

func init() {
	register(CustomIngressPluginName, &WalmPluginRunner{
		Run:  CustomIngressTransform,
		Type: Pre_Install,
	})
}

type AddIngressObject struct {
	Annotations map[string]string `json:"annotations" description:"ingress annotations"`
	Host        string            `json:"host" description:"ingress host"`
	Path        string            `json:"path" description:"ingress path"`
	ServiceName string            `json:"serviceName" description:"ingress backend service name"`
	ServicePort string            `json:"servicePort" description:"ingress backend service port"`
}

type CustomIngressArgs struct {
	IngressToAdd       map[string]*AddIngressObject `json:"ingressToAdd" description:"add extra ingress"`
	IngressToSkipNames []string                     `json:"ingressToSkipNames" description:"upgrade skip render ingress name"`
	IngressSkipAll     bool                         `json:"ingressSkipAll" description:"upgrade skip all ingress resources"`
}

func CustomIngressTransform(context *PluginContext, args string) (err error) {
	if args == "" {
		klog.Errorf("ignore ingress plugin, because plugin args is empty")
		return nil
	} else {
		klog.Infof("label pod args : %s", args)
	}

	customIngressArgs := &CustomIngressArgs{}
	err = json.Unmarshal([]byte(args), customIngressArgs)
	if err != nil {
		klog.Infof("failed to unmarshal plugin args : %s", err.Error())
		return err
	}

	for _, resource := range context.Resources {
		unstructuredObj := resource.(*unstructured.Unstructured)
		switch resource.GetObjectKind().GroupVersionKind().Kind {
		case "Ingress":
			if isSkippedIngress(unstructuredObj.GetName(), customIngressArgs) {
				err = addNestedStringMap(unstructuredObj.Object, map[string]string{ResourceUpgradePolicyAnno: UpgradePolicy}, "metadata", "annotations")
				if err != nil {
					klog.Errorf("failed add nested string map : %s", err.Error())
					return err
				}
			}
		}
	}

	for ingressName, addObj := range customIngressArgs.IngressToAdd {
		ingressObj, err := convertK8SIngress(context.R.Name, context.R.Namespace, ingressName, addObj)
		if err != nil {
			klog.Errorf("add ingress plugin error %v", err)
			continue
		}
		unstructuredObj, err := convertToUnstructured(ingressObj)
		if err != nil {
			klog.Infof("failed to convertToUnstructured : %v", *ingressObj)
			return err
		}
		context.Resources = append(context.Resources, unstructuredObj)
	}

	return
}

func isSkippedIngress(name string, args *CustomIngressArgs) bool {
	if args.IngressSkipAll == true {
		return true
	} else {
		for _, skipName := range args.IngressToSkipNames {
			if skipName == name {
				return true
			}
		}
		return false
	}
}

func convertK8SIngress(releaseName, releaseNamespace, ingressName string, addObj *AddIngressObject) (*v1beta1.Ingress, error) {
	if len(ingressName) == 0 || len(ingressName) > qualifiedNameMaxLength || !qualifiedNameRegexp.MatchString(ingressName) {
		return nil, errors.New(fmt.Sprintf("invaild ingress name %s", ingressName))
	}
	if !(strings.HasPrefix(addObj.Path, "/") && addObj.ServiceName != "" && addObj.ServicePort != "") {
		return nil, errors.New(fmt.Sprintf("invaild ingress object %v", *addObj))
	}

	ingressObj := &v1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind: "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("walmplugin-%s-%s-ingress", ingressName, releaseName),
			Namespace: releaseNamespace,
			Annotations: map[string]string{
				"transwarp/walmplugin.custom.ingress": "true",
				"kubernetes.io/ingress.class":         "nginx",
			},
			Labels: map[string]string{
				"release":  releaseName,
				"heritage": "walmplugin",
			},
		},
		Spec: v1beta1.IngressSpec{
			Rules: []v1beta1.IngressRule{
				{
					Host: addObj.Host,
					IngressRuleValue: v1beta1.IngressRuleValue{
						HTTP: &v1beta1.HTTPIngressRuleValue{
							Paths: []v1beta1.HTTPIngressPath{
								{
									Path: addObj.Path,
									Backend: v1beta1.IngressBackend{
										ServiceName: addObj.ServiceName,
										ServicePort: intstr.IntOrString{
											Type:   intstr.String,
											StrVal: addObj.ServicePort,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for k, v := range addObj.Annotations {
		ingressObj.ObjectMeta.Annotations[k] = v
	}

	return ingressObj, nil
}
