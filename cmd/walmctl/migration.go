package main

import (
	"WarpCloud/walm/cmd/walmctl/util/walmctlclient"
	"WarpCloud/walm/pkg/models/k8s"
	"encoding/json"
	corev1 "k8s.io/api/core/v1"

	k8sclient "WarpCloud/walm/pkg/k8s/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"path/filepath"
)

var longMigrateHelp = `
struct mig file template like
{
  "labels": {},
  "name": "string",
  "namespace": "string",
  "spec": {
    "namespace": "string",
    "podname": "string"
  },
  "srcHost": "string",
  "destHost": "string",
},

for example, migrate test1/pod1 from node1 to node2, you can specify like
{
  "name": "mig-test",
  "namespace": "mig-namespace",
  "spec": {
    "namespace": "test1",
    "podname": "pod1"
  },
  "srcHost": "node1",
  "destHost": "node2",
}
if schedule pods managed by sts on node1, you can specify easily like that
{
  "name": "mig-test",
  "namespace": "mig-namespace",
  "srcHost": "node1",
}


`

type migrateOptions struct {
	migType    string
	file       string
	kubeconfig string
	out        io.Writer
}

func newMigrationCmd(out io.Writer) *cobra.Command {
	migrate := &migrateOptions{out: out}

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "migrate node, pod --file",
		Long:  longMigrateHelp,
		RunE: func(cmd *cobra.Command, args []string) error {

			if args[0] != "pod" && args[0] != "node" {
				return errors.Errorf("unsupport type: %s, pod or node support only", args[0])
			}

			migrate.migType = args[0]
			return migrate.run()
		},
	}

	cmd.PersistentFlags().StringVarP(&migrate.file, "file", "f", "", "file to specify migs, required")
	cmd.PersistentFlags().StringVar(&migrate.kubeconfig, "kubeconfig", "kubeconfig", "kubeconfig path, required")
	cmd.MarkPersistentFlagRequired("file")
	cmd.MarkPersistentFlagRequired("kubeconfig")

	return cmd
}

func (migrate *migrateOptions) run() error {

	var mig k8s.Mig

	path, err := filepath.Abs(migrate.file)
	if err != nil {
		return errors.Errorf("failed to load migs specified file: %s", err.Error())
	}

	migByte, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Errorf("failed to read file %s : %s", path, err.Error())
	}

	err = json.Unmarshal(migByte, &mig)
	if err != nil {
		return errors.Errorf("failed to unmarshal mig file to struct k8sModel.Mig: %s", err.Error())
	}

	client := walmctlclient.CreateNewClient(walmserver)
	if err = client.ValidateHostConnect(); err != nil {
		return err
	}

	kubeconfig, err := filepath.Abs(migrate.kubeconfig)
	if err != nil {
		klog.Errorf("get kubeconfig failed: %s", err.Error())
		return err
	}

	k8sClient, err := k8sclient.NewClient("", kubeconfig)
	if err != nil {
		klog.Errorf("creates new Kubernetes Apiserver client failed: %s", err.Error())
		return err
	}

	if migrate.migType == "pod" {
		if mig.Spec.PodName == "" || mig.Spec.Namespace == ""{
			return errors.Errorf("spec.podname and spec.namespace must be set when migrate pod")
		}

		_, err = client.MigratePod(mig.Spec.Namespace, mig.Spec.PodName, mig)
		if err != nil {
			return errors.Errorf("failed to migrate pod: %s", err.Error())
		}
		return nil
	}

	if err = envPreCheck(k8sClient, mig.SrcHost, mig.DestHost); err != nil {
		klog.Errorf("env pre-check error: %s", err.Error())
		return err
	}

	podList, err := getPodListFromNode(k8sClient, mig.SrcHost)
	if err != nil {
		klog.Errorf("failed to get pods from node %s: %s", mig.SrcHost, err.Error())
		return err
	}

	prefixName := mig.Name
	for _, pod := range podList {
		mig.Labels = map[string]string{"migType": "node", "migName": prefixName}
		mig.Name = prefixName + "-" + pod.Namespace + "-" + pod.Name

		_, err = client.MigratePod(pod.Namespace, pod.Name, mig)
		if err != nil {
			return errors.Errorf("failed to migrate pod: %s", err.Error())
		}
		return nil
	}

	// Todo: cordon pod

	// Todo: migrate progress

	// Todo: failed retry

	// Todo: return migrate status

	// Todo: failed notification

	return nil
}

func getPodListFromNode(k8sClient *kubernetes.Clientset, srcHost string) ([]corev1.Pod, error) {
	podList := &corev1.PodList{
		Items: []corev1.Pod{},
	}
	pods, err := k8sClient.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		klog.Errorf("failed to list pods: %s", err.Error())
		return nil, err
	}

	for _, pod := range pods.Items {
		if pod.Spec.NodeName == srcHost {
			for _, ownerReference := range pod.OwnerReferences {
				if ownerReference.Kind == "StatefulSet" {
					podList.Items = append(podList.Items, pod)
				}
			}
		}
	}
	return podList.Items, nil
}

// Todo: Env Pre-Check
func envPreCheck(k8sClient *kubernetes.Clientset, srcHost string, destHost string) error {

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

	// Todo: other check

	return nil
}
