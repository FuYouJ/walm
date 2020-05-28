package converter

import (
	"WarpCloud/walm/pkg/models/k8s"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"
	"net"
	"strconv"
)

func ConvertServiceFromK8s(oriService *corev1.Service, endpoints *corev1.Endpoints) (walmService *k8s.Service, err error) {
	if oriService == nil {
		return
	}
	service := oriService.DeepCopy()

	walmService = &k8s.Service{
		Meta:        k8s.NewMeta(k8s.ServiceKind, service.Namespace, service.Name, k8s.NewState("Ready", "", "")),
		Labels:      service.Labels,
		ExternalIPs: service.Spec.ExternalIPs,
		ResourceVersion: service.ResourceVersion,
		Selector:    service.Spec.Selector,
		ClusterIp:   service.Spec.ClusterIP,
		ServiceType: string(service.Spec.Type),
		Annotations: service.Annotations,
	}

	if walmService.Annotations == nil {
		walmService.Annotations = map[string]string{}
	}

	walmService.Ports, err = buildWalmServicePorts(service, endpoints)
	if err != nil {
		klog.Errorf("failed to build walm service ports: %s", err.Error())
		return
	}

	return
}

func ConvertServiceToK8s(walmService *k8s.Service) (k8sService *corev1.Service, err error) {
	if walmService == nil {
		return nil, nil
	}
	var serviceType corev1.ServiceType
	switch walmService.ServiceType {
	case "ClusterIP":
		serviceType = corev1.ServiceTypeClusterIP
	case "NodePort":
		serviceType = corev1.ServiceTypeNodePort
	case "LoadBalancer":
		serviceType = corev1.ServiceTypeLoadBalancer
	case "ExternalName":
		serviceType = corev1.ServiceTypeExternalName
	case "":
	default:
		return nil, errors.Errorf("invalid service type %s", walmService.ServiceType)
	}

	var servicePorts []corev1.ServicePort
	for _, port := range walmService.Ports {
		var protocol corev1.Protocol
		switch port.Protocol {
		case "TCP", "":
			protocol = corev1.ProtocolTCP
		case "UDP":
			protocol = corev1.ProtocolUDP
		case "SCTP":
			protocol = corev1.ProtocolSCTP
		default:
			return nil, errors.Errorf("invalid service port protocol %s", port.Protocol)

		}
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:     port.Name,
			Protocol: protocol,
			Port:     port.Port,
			TargetPort: intstr.IntOrString{
				Type:   intstr.String,
				StrVal: port.TargetPort,
			},
			NodePort: port.NodePort,
		})
	}

	k8sService = &corev1.Service{
		TypeMeta: v1.TypeMeta{
			Kind: string(k8s.ServiceKind),
		},
		ObjectMeta: v1.ObjectMeta{
			ResourceVersion: walmService.ResourceVersion,
			Labels:    walmService.Labels,
			Annotations: walmService.Annotations,
			Name:      walmService.Name,
			Namespace: walmService.Namespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: walmService.ClusterIp,
			Selector: walmService.Selector,
			Type: serviceType,
			ExternalIPs: walmService.ExternalIPs,
			Ports: servicePorts,
		},
	}
	return k8sService, nil
}

func buildWalmServicePorts(service *corev1.Service, endpoints *corev1.Endpoints) ([]k8s.ServicePort, error) {
	ports := []k8s.ServicePort{}
	for _, port := range service.Spec.Ports {
		walmServicePort := k8s.ServicePort{
			Name:       port.Name,
			Port:       port.Port,
			NodePort:   port.NodePort,
			Protocol:   string(port.Protocol),
			TargetPort: port.TargetPort.String(),
		}
		if endpoints != nil {
			walmServicePort.Endpoints = formatEndpoints(endpoints, sets.NewString(port.Name))
		} else {
			walmServicePort.Endpoints = []string{}
		}
		ports = append(ports, walmServicePort)
	}

	return ports, nil
}

func formatEndpoints(endpoints *corev1.Endpoints, ports sets.String) (list []string) {
	list = []string{}
	for i := range endpoints.Subsets {
		ss := &endpoints.Subsets[i]
		for i := range ss.Ports {
			port := &ss.Ports[i]
			if ports == nil || ports.Has(port.Name) {
				for i := range ss.Addresses {
					addr := &ss.Addresses[i]
					hostPort := net.JoinHostPort(addr.IP, strconv.Itoa(int(port.Port)))
					list = append(list, hostPort)
				}
			}
		}
	}

	return
}
