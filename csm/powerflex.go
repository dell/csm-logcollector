package csm

import (
	"bytes"
	"context"
	utils "csm-logcollector/utils"
	"fmt"
	"io"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	describe "k8s.io/kubectl/pkg/describe"
)

// Logging object
var pflxLog = utils.GetLogger()

// PowerFlexStruct for PowerFlex platform
type PowerFlexStruct struct {
	StorageNameSpaceStruct
}

func runningpodsPowerflex(namespaceDirectoryName string, pod *corev1.Pod) {
	var dirName string
	fmt.Printf("\tpod.Name........%s\n", pod.Name)
	dirName = namespaceDirectoryName + "/" + pod.Name
	podDirectoryName := createDirectory(dirName)
	fmt.Printf("\tThere are %d containers for this pod\n", len(pod.Spec.Containers))
	if len(pod.Spec.Containers) > 1 {
		// check for sdc-monitor sidecar in node pod
		if strings.Contains(pod.Name, "node") {
			var flag bool
			flag = false
			for container := range pod.Spec.Containers {
				if pod.Spec.Containers[container].Name == "sdc-monitor" {
					flag = true
				}
			}
			if flag {
				fmt.Printf("\tsdc-monitor container is deployed as monitor value is set in myvalues.yaml")
			} else {
				fmt.Printf("\tsdc-monitor container is not deployed as monitor value is not set in myvalues.yaml")
			}
		}
		for container := range pod.Spec.Containers {
			fmt.Println("\t\t", pod.Spec.Containers[container].Name)
			dirName = podDirectoryName + "/" + pod.Spec.Containers[container].Name
			containerDirectoryName := createDirectory(dirName)

			opts := corev1.PodLogOptions{}
			opts.Container = pod.Spec.Containers[container].Name
			req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &opts)
			podLogs, err := req.Stream(context.TODO())
			if err != nil {
				pflxLog.Errorf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
				fmt.Printf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
			}
			defer podLogs.Close()
			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, podLogs)
			if err != nil {
				pflxLog.Errorf("Error in copy information from podLogs to buf: %s", err.Error())
				fmt.Printf("Error in copy information from podLogs to buf: %s", err.Error())
			}
			str := buf.String()
			filename := pod.Name + "-" + pod.Spec.Containers[container].Name + ".txt"
			fmt.Println("\tLOG collected in file..........")
			captureLOG(containerDirectoryName, filename, str)
		}
	} else {
		dirName = podDirectoryName + "/" + pod.Spec.Containers[0].Name
		containerDirectoryName := createDirectory(dirName)

		req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
		podLogs, err := req.Stream(context.TODO())
		if err != nil {
			pflxLog.Errorf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
			fmt.Printf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
		}
		defer podLogs.Close()
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		if err != nil {
			pflxLog.Errorf("Error in copy information from podLogs to buf: %s", err.Error())
			fmt.Printf("Error in copy information from podLogs to buf: %s", err.Error())
		}
		str := buf.String()
		filename := pod.Name + ".txt"
		fmt.Println("LOG collected in file..........")
		captureLOG(containerDirectoryName, filename, str)
	}
}

func nonrunningpodsPowerflex(namespaceDirectoryName string, pod *corev1.Pod) {
	var dirName string
	fmt.Printf("\tpod.Name........%s\n", pod.Name)
	dirName = namespaceDirectoryName + "/" + pod.Name
	podDirectoryName := createDirectory(dirName)
	fmt.Printf("\tThere are %d containers for this pod\n", len(pod.Spec.Containers))
	if len(pod.Spec.Containers) > 1 {
		// check for sdc-monitor sidecar in node pod
		if strings.Contains(pod.Name, "node") {
			var flag bool
			flag = false
			for container := range pod.Spec.Containers {
				if pod.Spec.Containers[container].Name == "sdc-monitor" {
					flag = true
				}
			}
			if flag {
				pflxLog.Infof("sdc-monitor container is deployed as monitor value is set in myvalues.yaml")
				fmt.Printf("\tsdc-monitor container is deployed as monitor value is set in myvalues.yaml")
			} else {
				pflxLog.Infof("sdc-monitor container is not deployed as monitor value is not set in myvalues.yaml")
				fmt.Printf("\tsdc-monitor container is not deployed as monitor value is not set in myvalues.yaml")
			}
		}
		for container := range pod.Spec.Containers {
			fmt.Println("\t\t", pod.Spec.Containers[container].Name)
			dirName = podDirectoryName + "/" + pod.Spec.Containers[container].Name
			containerDirectoryName := createDirectory(dirName)
			var str string = "Pod status: " + string(pod.Status.Phase)
			filename := pod.Name + ".txt"
			fmt.Println("\tLOG collected in file..........")
			captureLOG(containerDirectoryName, filename, str)
		}
	} else {
		dirName = podDirectoryName + "/" + pod.Spec.Containers[0].Name
		containerDirectoryName := createDirectory(dirName)
		var str string = "Pod status: " + string(pod.Status.Phase)
		filename := pod.Name + ".txt"
		fmt.Println("LOG collected in file..........")
		captureLOG(containerDirectoryName, filename, str)
	}
}

// GetLogs accesses the API to get driver/sidecarpod logs of RUNNING pods
func (p PowerFlexStruct) GetLogs(namespace string, optionalFlag string) {
	clientset := GetClientSetFromConfig()
	p.namespaceName, _, _ = p.GetDriverDetails(namespace)
	fmt.Println("\n*******************************************************************************")
	GetNodes()
	nsarray := GetNamespaces()
	p.ValidateNamespace(nsarray)
	podarray := p.GetPods()

	var dirName string
	t := time.Now().Format("20060102150405") //YYYYMMDDhhmmss
	dirName = namespace + "_" + t
	namespaceDirectoryName := createDirectory(dirName)

	for i := 0; i < len(podarray); i++ {
		dirName = namespaceDirectoryName + "/" + podarray[i]
		podDirectoryName := createDirectory(dirName)
		p.DescribePods(podarray[i], describe.DescriberSettings{ShowEvents: true}, podDirectoryName)
	}

	p.GetLeaseDetails()
	fmt.Printf("\nOptional flag: %s", optionalFlag)
	fmt.Println("\nCollecting Running Pod Logs (driver logs, sidecar logs)")

	podallns, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		pflxLog.Errorf("Getting all pods failed with error: %s", err.Error())
		panic(err.Error())
	}
	for _, pod := range podallns.Items {
		if pod.Namespace == namespace {
			if pod.Status.Phase == "Running" {
				runningpodsPowerflex(namespaceDirectoryName, &pod)
				fmt.Println("\t*************************************************************")
				pflxLog.Infof("Logs collected for runningpods of %s", namespace)
			} else {
				nonrunningpodsPowerflex(namespaceDirectoryName, &pod)
				fmt.Println("\t*************************************************************")
				pflxLog.Infof("Logs collected for non-runningpods of %s", namespace)
			}
		}
	}

	errMsg := createTarball(namespaceDirectoryName, ".")

	if errMsg != nil {
		pflxLog.Errorf("Creating tarball %s failed with error: %s", namespaceDirectoryName, errMsg.Error())
		panic(errMsg.Error())
	}
}
