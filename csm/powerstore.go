package csm

import (
	"bytes"
	"context"
	utils "csm-logcollector/utils"
	"fmt"
	"io"
	"strings"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	describe "k8s.io/kubectl/pkg/describe"
)

// Logging object
var psLog = utils.GetLogger()

// PowerStoreStruct for PowerStore platform
type PowerStoreStruct struct {
	StorageNameSpaceStruct
}

// GetLeaseDetails collects lease details
func (p PowerStoreStruct) GetLeaseDetails(namespace string) {
	fmt.Printf("\n\nLease pod for %s..............\n", namespace)
	fmt.Println("=====================================")
	_ = &coordinationv1.Lease{}
	leasePodList, err := clientset.CoordinationV1().Leases(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		psLog.Errorf("Getting lease details in namespace %s failed with error: %s", namespace, err.Error())
		panic(err.Error())
	}
	leasepod := "external-attacher-leader-" + namespace + "-dellemc-com"

	for _, lease := range leasePodList.Items {
		if strings.Contains(lease.Name, leasepod) {
			fmt.Printf("\t%s\n", lease.Name)
			fmt.Printf("\t%s\n", lease.Namespace)
			fmt.Printf("\t%s\n", *lease.Spec.HolderIdentity) // Points to same controller pod for all instances
			psLog.Debugf("Lease pod detailes: %s, %s, %s", lease.Name, lease.Namespace, *lease.Spec.HolderIdentity)
			fmt.Println()
		}
	}
}

func runningpodsPowerstore(namespaceDirectoryName string, pod *corev1.Pod) {
	var dirName string
	fmt.Printf("pod.Name........%s\n", pod.Name)
	fmt.Printf("pod.Status.Phase.......%s\n", pod.Status.Phase)
	dirName = namespaceDirectoryName + "/" + pod.Name
	podDirectoryName := createDirectory(dirName)
	fmt.Printf("There are %d containers for the pod\n", len(pod.Spec.Containers))
	if len(pod.Spec.Containers) > 1 {
		for container := range pod.Spec.Containers {
			fmt.Println("\t", pod.Spec.Containers[container].Name)
			dirName = podDirectoryName + "/" + pod.Spec.Containers[container].Name
			containerDirectoryName := createDirectory(dirName)

			opts := corev1.PodLogOptions{}
			opts.Container = pod.Spec.Containers[container].Name
			req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &opts)
			podLogs, err := req.Stream(context.TODO())
			if err != nil {
				psLog.Errorf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
				fmt.Printf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
			}
			defer podLogs.Close()
			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, podLogs)
			if err != nil {
				psLog.Errorf("Error in copy information from podLogs to buf: %s", err.Error())
				fmt.Printf("Error in copy information from podLogs to buf: %s", err.Error())
			}
			str := buf.String()

			filename := pod.Name + "-" + pod.Spec.Containers[container].Name + ".txt"
			captureLOG(containerDirectoryName, filename, str)
		}
		fmt.Println()
	} else {
		dirName = podDirectoryName + "/" + pod.Spec.Containers[0].Name
		containerDirectoryName := createDirectory(dirName)

		req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
		podLogs, err := req.Stream(context.TODO())
		if err != nil {
			psLog.Errorf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
			fmt.Printf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
		}
		defer podLogs.Close()
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		if err != nil {
			psLog.Errorf("Error in copy information from podLogs to buf: %s", err.Error())
			fmt.Printf("Error in copy information from podLogs to buf: %s", err.Error())
		}
		str := buf.String()

		filename := pod.Name + ".txt"
		captureLOG(containerDirectoryName, filename, str)
		fmt.Println()
	}
}

func nonrunningpodsPowerstore(namespaceDirectoryName string, pod *corev1.Pod) {
	var dirName string
	fmt.Printf("pod.Name........%s\n", pod.Name)
	fmt.Printf("pod.Status.Phase.......%s\n", pod.Status.Phase)
	fmt.Printf("There are %d containers for the pod\n", len(pod.Spec.Containers))
	dirName = namespaceDirectoryName + "/" + pod.Name
	podDirectoryName := createDirectory(dirName)
	if len(pod.Spec.Containers) > 1 {
		for container := range pod.Spec.Containers {
			fmt.Println("\t", pod.Spec.Containers[container].Name)
			dirName = podDirectoryName + "/" + pod.Spec.Containers[container].Name
			containerDirectoryName := createDirectory(dirName)
			var str string = "Pod status: not running"
			filename := pod.Name + ".txt"
			captureLOG(containerDirectoryName, filename, str)
			fmt.Println()
		}
	} else {
		dirName = podDirectoryName + "/" + pod.Spec.Containers[0].Name
		containerDirectoryName := createDirectory(dirName)
		var str string = "Pod status: not running"

		filename := pod.Name + ".txt"
		captureLOG(containerDirectoryName, filename, str)
		fmt.Println()
	}
}

// GetLogs accesses the API to get driver/sidecarpod logs of RUNNING pods
func (p PowerStoreStruct) GetLogs(namespace string, optionalFlag string) {
	clientset := GetClientSetFromConfig()
	fmt.Println("\n*******************************************************************************")
	p.GetNodes()
	nsarray := p.GetNamespaces()
	p.ValidateNamespace(nsarray, namespace)
	podarray := p.GetPods(namespace)

	var dirName string
	t := time.Now().Format("20060102150405") //YYYYMMDDhhmmss
	dirName = namespace + "_" + t
	namespaceDirectoryName := createDirectory(dirName)

	for i := 0; i < len(podarray); i++ {
		dirName = namespaceDirectoryName + "/" + podarray[i]
		podDirectoryName := createDirectory(dirName)
		p.DescribePods(namespace, podarray[i], describe.DescriberSettings{ShowEvents: true}, podDirectoryName)
	}

	p.GetDriverDetails(namespace)
	p.GetLeaseDetails(namespace)
	// access the API to get driver/sidecarpod logs of RUNNING pods

	fmt.Printf("Optional flag: %s\n", optionalFlag)
	fmt.Println("\n\nCollecting RUNNING POD LOGS (driver logs, sidecar logs)..........")

	podallns, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		psLog.Errorf("Getting all pods failed with error: %s", err.Error())
		panic(err.Error())
	}
	for _, pod := range podallns.Items {
		if pod.Namespace == namespace {
			if pod.Status.Phase == "Running" {
				runningpodsPowerstore(namespaceDirectoryName, &pod)
			} else {
				nonrunningpodsPowerstore(namespaceDirectoryName, &pod)
			}
		}
	}

	errMsg := createTarball(namespaceDirectoryName, ".")

	if errMsg != nil {
		psLog.Errorf("Creating tarball %s failed with error: %s", namespaceDirectoryName, errMsg.Error())
		panic(errMsg.Error())
	}
}
