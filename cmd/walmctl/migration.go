package main

import (
	"WarpCloud/walm/cmd/walmctl/util/walmctlclient"
	k8sclient "WarpCloud/walm/pkg/k8s/client"
	"WarpCloud/walm/pkg/k8s/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"path/filepath"
)

var longMigrateHelp = `
1. migrate one pod to another node
2.migrate all pods managed by statefulsets of one node to another node;


`

type migrateOptions struct {
	srcHost      string
	destHost     string
	migType      string
	migName      string
	migNamespace string
	podName      string
	podNamespace string
	kubeconfig   string
	out          io.Writer
}

func newMigrationCmd(out io.Writer) *cobra.Command {
	migrate := &migrateOptions{out: out}

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "migrate node nodeName, migrate pod podName",
		Long:  longMigrateHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.Errorf("use migrate node/pod nodeName/podName instead")
			}

			switch args[0] {
			case "node":
				migrate.srcHost = args[1]
			case "pod":
				migrate.podName = args[1]
			default:
				return errors.Errorf("unsupport type: %s, pod or node support only", args[0])
			}
			migrate.migType = args[0]

			return migrate.run()
		},
	}
	cmd.PersistentFlags().StringVar(&migrate.migName, "migName", "", "name for migration crd, required")
	cmd.PersistentFlags().StringVar(&migrate.migNamespace, "migNamespace", "", "namespace to store migration crd, not required")
	cmd.PersistentFlags().StringVar(&migrate.podNamespace, "podNamespace", "", "pod namespace")
	cmd.PersistentFlags().StringVar(&migrate.destHost, "destHost", "", "node you want to migrate to, not required")
	cmd.PersistentFlags().StringVar(&migrate.kubeconfig, "kubeconfig", "kubeconfig", "kubeconfig path, required")
	cmd.MarkPersistentFlagRequired("migName")
	cmd.MarkPersistentFlagRequired("kubeconfig")

	return cmd
}

func (migrate *migrateOptions) run() error {
	var (
		err error
		//resp *resty.Response
	)

	client := walmctlclient.CreateNewClient(walmserver)
	if err = client.ValidateHostConnect(); err != nil {
		return err
	}

	if migrate.migType == "pod" {
		_, err = client.MigratePod(migrate.podNamespace, migrate.podName, migrate.migName, migrate.migNamespace, migrate.destHost)
		if err != nil {
			return errors.Errorf("failed to migrate pod: %s", err.Error())
		}
		return nil
	}

	migrate.kubeconfig, err = filepath.Abs(migrate.kubeconfig)
	if err != nil {
		klog.Errorf("get kubeconfig failed: %s", err.Error())
		return err
	}

	k8sClient, err := k8sclient.NewClient("", migrate.kubeconfig)
	if err != nil {
		klog.Errorf("creates new Kubernetes Apiserver client failed: %s", err.Error())
		return err
	}

	if err = envPreCheck(k8sClient, migrate.srcHost, migrate.destHost); err != nil {
		klog.Errorf("env pre-check error: %s", err.Error())
		return err
	}

	podList, err := getPodListFromNode(k8sClient, migrate.srcHost)
	if err != nil {
		klog.Errorf("failed to get pods from node %s: %s", migrate.srcHost,  err.Error())
		return err
	}


	// Todo: cordon pod

	// Todo: migrate progress

	// Todo: failed retry

	// Todo: return migrate status

    // Todo: failed notification
	for _, pod := range podList {
		_, err := client.MigratePod(pod.Namespace, pod.Name, migrate.migName, migrate.migNamespace, migrate.destHost)
		if err != nil {
			klog.Errorf("failed to migrate pod: %s", err.Error())
			return err
		}
	}
	return nil
}

func getPodListFromNode(k8sClient *kubernetes.Clientset, srcHost string) ([]v1.Pod, error) {

	var pods []v1.Pod

	// get namespace
	namespaceList, err := k8sClient.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		klog.Errorf("failed to get namespace from node %s: %s", srcHost, err.Error())
		return nil, err
	}

	for _, namespace := range namespaceList.Items {

		// get all sts from namespace
		stsList, err := k8sClient.AppsV1().StatefulSets(namespace.Name).List(metav1.ListOptions{})
		if err != nil {
			klog.Errorf("failed to get statefulsets from namespace %s: %s", namespace.Name, err.Error())
			return nil, err
		}

		// get node's pod from sts
		for _, sts := range stsList.Items {

			labelSelector, err := utils.ConvertLabelSelectorToStr(sts.Spec.Selector)
			if err != nil {
				klog.Errorf("failed to convert statefulset %s labelselector to str: %s", sts.Name, err.Error())
				return nil, err
			}
			podList, err := k8sClient.CoreV1().Pods(namespace.Name).List(metav1.ListOptions{
				LabelSelector: labelSelector,
			})
			if err != nil {
				klog.Errorf("failed to get pods from statefulsets %s: %s", sts.Name, err.Error())
				return nil, err
			}

			for _, pod := range podList.Items {
				if pod.Spec.NodeName != srcHost {
					continue
				}
				pods = append(pods, pod)
			}
		}
	}
	return pods, nil
}

// Todo: Env Pre-Check
func envPreCheck(k8sClient *kubernetes.Clientset, srcHost string, destHost string) error{

	// Node Check
	nodeList, err := k8sClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		klog.Errorf("failed to get nodes: %s", err.Error())
		return err
	}
	if len(nodeList.Items) < 2 {
		return errors.Errorf("only one node, migration make no sense")
	}

	_, err = k8sClient.CoreV1().Nodes().Get(srcHost, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get node %s: %s", srcHost, err.Error())
		return err
	}

	if destHost != "" {
		_, err = k8sClient.CoreV1().Nodes().Get(destHost, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get node %s: %s", srcHost, err.Error())
			return err
		}
	}

	// Crd Check
	// Todo: other check

	return nil
}
