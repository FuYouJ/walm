package main

import (
	"WarpCloud/walm/cmd/walmctl/util/walmctlclient"
	k8sclient "WarpCloud/walm/pkg/k8s/client"
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	"encoding/json"
	"fmt"
	"github.com/migration/pkg/apis/tos/v1beta1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var longMigrateHelp = `
To ensure migrate node work, you may set $KUBE_CONFIG or --kubeconfig while using migrate cmd.
Using:
	"export $KUBE_CONFIG=..." or "walmctl migrate ....  --kubeconfig ..."

migrate pod: kubectl -s x.x.x.x:y migrate pod podname
migrate node:
	In cluster: kubectl -s x.x.x.x:y migrate node nodeName
    Out cluster: --kubeconfig
`

type migrateOptions struct {
	kind       string
	name       string
	kubeconfig string
	destNode   string
	out        io.Writer
}

func newMigrationCmd(out io.Writer) *cobra.Command {
	migrate := &migrateOptions{out: out}

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "migrate node, pod",
		Long:  longMigrateHelp,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 2 {
				return errors.New("arguments error, migrate node/pod nodeName/podName")
			}

			if args[0] != "pod" && args[0] != "node" {
				return errors.Errorf("unsupport kind: %s, pod or node support only", args[0])
			}

			migrate.kind = args[0]
			migrate.name = args[1]
			return migrate.run()
		},
	}
	cmd.PersistentFlags().StringVar(&migrate.destNode, "destNode", "", "dest node to migrate")
	cmd.PersistentFlags().StringVar(&migrate.kubeconfig, "kubeconfig", os.Getenv("KUBE_CONFIG"), "k8s cluster config")

	return cmd
}

func (migrate *migrateOptions) run() error {
	var err error
	if walmserver == "" {
		return errServerRequired
	}

	client := walmctlclient.CreateNewClient(walmserver)
	if err = client.ValidateHostConnect(); err != nil {
		return err
	}

	if migrate.kind == "pod" {
		err = migratePodPreCheck(client, namespace, migrate.name, migrate.destNode)
		if err != nil {
			return err
		}

		_, err = client.MigratePod(namespace, &k8sModel.PodMigRequest{
			PodName:     migrate.name,
			DestNode: migrate.destNode,
		})
		if err != nil {
			return errors.Errorf("failed to migrate pod: %s", err.Error())
		}

		fmt.Printf("create pod migrate task succeed.\n")
		return nil
	}

	if migrate.kubeconfig == "" {
		fmt.Println("[WARNING]: Neither --kubeconfig nor ENV KUBE_CONFIG was specified.  Using the inClusterConfig.  This might not work when migrate node.")
	} else {
		migrate.kubeconfig, err = filepath.Abs(migrate.kubeconfig)
		if err != nil {
			klog.Errorf("failed to get kubeconfig path: %s", err.Error())
			return err
		}
	}
	k8sClient, err := k8sclient.NewClient("", migrate.kubeconfig)
	if err != nil {
		klog.Errorf("Failed to create new kubernetes client %s", err.Error())
		return err
	}

	if err = envPreCheck(k8sClient, migrate.name, migrate.destNode); err != nil {
		klog.Errorf("env pre-check error: %s", err.Error())
		return err
	}

	podList, err := getPodListFromNode(k8sClient, migrate.name)
	if err != nil {
		klog.Errorf("failed to get pods from node %s: %s", migrate.name, err.Error())
		return err
	}

	/* check all pods ready to be migrated */
	for _, pod := range podList {
		err = migratePodPreCheck(client, pod.Namespace, pod.Name, migrate.destNode)
		if err != nil {
			return err
		}
	}

	for _, pod := range podList {
		_, err = client.MigratePod(pod.Namespace, &k8sModel.PodMigRequest{
			PodName:     pod.Name,
			DestNode: migrate.destNode,
			Labels:   map[string]string{"migType": "node", "srcNode": migrate.name},
		})
		if err != nil {
			klog.Errorf("send migrate pod request failed: %s", err.Error())
			return err
		}
	}

	fmt.Printf("create node migrate task succeed.\n")

	migStatus, errMsgs, err := getMigDetails(client, migrate.name)
	if err != nil {
		klog.Errorf("failed to get node migration response: %s", err.Error())
		return err
	}

	for i := 0; i < 60; i++ {
		progress := "[" + bar(migStatus.Succeed, migStatus.Total) + "]" + strconv.Itoa(migStatus.Succeed) + " / " + strconv.Itoa(migStatus.Total)
		fmt.Printf("\r%s", progress)
		time.Sleep(30 * time.Second)
		migStatus, errMsgs, err = getMigDetails(client, migrate.name)
		if err != nil {
			klog.Errorf("failed to get node migration response: %s", err.Error())
			return err
		}
		if migStatus.Succeed == migStatus.Total || len(errMsgs) + migStatus.Succeed == migStatus.Total{
			break
		}
	}
	fmt.Println()
	if len(errMsgs) > 0 {
		for _, errMsg := range errMsgs {
			fmt.Printf("[Error]: %s\n", errMsg)
		}
		if len(errMsgs) + migStatus.Succeed < migStatus.Total {
			return errors.Errorf("migrate node timeout, check the error messages above")
		}
		return errors.Errorf("migrate node failed, check the error messages above")
	}
	if migStatus.Succeed == migStatus.Total {
		fmt.Printf("migration node succeed, use `walmctl get migration node nodeName` for detail information")
	} else {
		return errors.Errorf("migration node timeout, use `walmctl get migration node nodeName` for later information")
	}
	return nil
}

func getMigDetails(client *walmctlclient.WalmctlClient, node string) (k8sModel.MigStatus, []string, error){
	var migStatus k8sModel.MigStatus
	var errMsgs []string
	resp, err := client.GetNodeMigration(node)
	if err != nil {
		klog.Errorf("failed to get node migration: %s", err.Error())
		return migStatus, errMsgs, err
	}
	err = json.Unmarshal(resp.Body(), &migStatus)
	if err != nil {
		klog.Errorf("failed to unmarshal node migrate response: %s", err.Error())
		return migStatus, errMsgs, err
	}
	for _, item := range migStatus.Items {
		if item.State.Status == v1beta1.MIG_FAILED {
			errMsgs = append(errMsgs, item.State.Message)
		}
	}
	return migStatus, errMsgs, nil
}

func bar(count, size int) string {
	str := ""
	for i := 0; i < size; i++ {
		if i < count {
			str += "="
		} else {
			str += " "
		}
	}
	return str
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

func envPreCheck(k8sClient *kubernetes.Clientset, srcHost string, destHost string) error {

	nodeList, err := k8sClient.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		klog.Errorf("failed to get nodes: %s", err.Error())
		return err
	}
	if len(nodeList.Items) < 2 {
		return errors.Errorf("only one node, migration make no sense")
	}

	srcNode, err := k8sClient.CoreV1().Nodes().Get(srcHost, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get node %s: %s", srcHost, err.Error())
		return err
	}

	if destHost != "" {
		destNode, err := k8sClient.CoreV1().Nodes().Get(destHost, metav1.GetOptions{})
		if err != nil {
			klog.Errorf("failed to get node %s: %s", srcHost, err.Error())
			return err
		}
		if destNode.Spec.Unschedulable {
			return errors.Errorf("dest node is Unschedulable, run `kubectl uncordon ...`")
		}
	}

	/* cordon node */
	if srcNode.Spec.Unschedulable == false {
		oldData, err := json.Marshal(srcNode)
		if err != nil {
			return err
		}

		srcNode.Spec.Unschedulable = true
		newData, err := json.Marshal(srcNode)
		if err != nil {
			return err
		}
		patchBytes, patchErr := strategicpatch.CreateTwoWayMergePatch(oldData, newData, srcNode)
		if patchErr == nil {
			_, err = k8sClient.CoreV1().Nodes().Patch(srcNode.Name, types.StrategicMergePatchType, patchBytes)

		} else {
			_, err = k8sClient.CoreV1().Nodes().Update(srcNode)
		}
		if err != nil {
			fmt.Printf("error: unable to cordon node %q: %v\n", srcNode.Name, err)
		}
	} else {
		klog.Infof("node %s is unschedulable now", srcNode.Name)
	}
	return nil
}

func migratePodPreCheck(client *walmctlclient.WalmctlClient, namespace string, name string, destNode string) error {
	if namespace == "" {
		return errNamespaceRequired
	}
	resp, err := client.GetPodMigration(namespace, name)
	if err != nil {
		if strings.Contains(err.Error(), "not found error") {
			klog.Warningf("pod migration not found, skipped")
		} else {
			return errors.Errorf("failed to get pod migration: %s", err.Error())
		}
	}
	var podMig k8sModel.Mig
	if resp != nil {
		err = json.Unmarshal(resp.Body(), &podMig)
		if err != nil {
			klog.Errorf("failed to unmarshal response body to pod migration status")
			return err
		}
		switch podMig.State.Status {
		case v1beta1.MIG_CREATED, v1beta1.MIG_IN_PROGRESS, "":
			return errors.Errorf("Pod %s/%s migration in progress, please wait for the last process end.", podMig.Spec.Namespace, podMig.Spec.PodName)
		case v1beta1.MIG_FAILED:
			_, err = client.DeletePodMigration(podMig.Spec.Namespace, podMig.Spec.PodName)
			if err != nil {
				klog.Errorf("failed to delete last failed pod migration %s: %s", podMig.Name, err.Error())
				return err
			}
			return errors.Errorf("Last migration for pod failed: %s.\nWe already help you delete the failed pod migration, please fix and try!", podMig.State.Message)
		case v1beta1.MIG_FINISH:
			_, err = client.DeletePodMigration(podMig.Spec.Namespace, podMig.Spec.PodName)
			if err != nil {
				klog.Errorf("failed to delete pod migration: %s", err.Error())
				return err
			}
		}
	}

	return nil
}
