/*
 Copyright (c) 2021 Dell Inc, or its subsidiaries.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
      http://www.apache.org/licenses/LICENSE-2.0
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

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
var pflxLog, _ = utils.GetLogger()

// PowerFlexStruct for PowerFlex platform
type PowerFlexStruct struct {
	StorageNameSpaceStruct
}

// GetRunningPods is overridden for PowerFlex specific implementation
func (p PowerFlexStruct) GetRunningPods(namespaceDirectoryName string, pod *corev1.Pod, dateRange *metav1.Time, optionalFlag string) {
	var dirName string
	fmt.Printf("pod.Name........%s\n", pod.Name)
	fmt.Printf("pod.Status.Phase.......%s\n", pod.Status.Phase)
	dirName = namespaceDirectoryName + "/" + pod.Name
	podDirectoryName := createDirectory(dirName)

	// check for sdc-monitor sidecar in node pod
	if strings.Contains(pod.Name, "node") {
		var flag bool = false
		for container := range pod.Spec.Containers {
			if pod.Spec.Containers[container].Name == "sdc-monitor" {
				flag = true
				break
			}
		}
		pflxLog.Infof("sdc-monitor container is deployed successfully: %t for %s", flag, pod.Name)
	}

	if optionalFlag == "False" || optionalFlag == "false" {
		str := "Pod " + pod.Name + " is in running state\n"
		filename := pod.Name + ".txt"
		captureLOG(podDirectoryName, filename, str)
		fmt.Println()
	} else {
		for container := range pod.Spec.Containers {
			fmt.Printf("\t Collecting Logs from container %s\n", pod.Spec.Containers[container].Name)
			dirName = podDirectoryName + "/" + pod.Spec.Containers[container].Name
			containerDirectoryName := createDirectory(dirName)

			opts := corev1.PodLogOptions{}
			opts.Container = pod.Spec.Containers[container].Name
			if dateRange != nil {
				fmt.Printf("Logs will be collected from: %v\n", dateRange)
				opts.SinceTime = dateRange
			}
			req := clientset.CoreV1().Pods(p.namespaceName).GetLogs(pod.Name, &opts)
			podLogs, err := req.Stream(context.TODO())
			if err != nil {
				pflxLog.Errorf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
			}

			defer func() {
				if err := podLogs.Close(); err != nil {
					pflxLog.Fatalf("Error streaming file with error %s \n", err.Error())
				}
			}()

			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, podLogs)
			if err != nil {
				pflxLog.Errorf("Error in copy information from podLogs to buf: %s", err.Error())
			}
			str := buf.String()

			filename := pod.Name + "-" + pod.Spec.Containers[container].Name + ".txt"
			captureLOG(containerDirectoryName, filename, str)
		}
	}
}

// GetNonRunningPods is overridden for PowerFlex specific implementation
func (p PowerFlexStruct) GetNonRunningPods(namespaceDirectoryName string, pod *corev1.Pod) {
	var dirName string
	fmt.Printf("pod.Name........%s\n", pod.Name)
	fmt.Printf("pod.Status.Phase.......%s\n", pod.Status.Phase)
	dirName = namespaceDirectoryName + "/" + pod.Name
	podDirectoryName := createDirectory(dirName)
	containerCount := len(pod.Spec.Containers)
	fmt.Printf("\tThere are %d containers for this pod\n", containerCount)

	// check for sdc-monitor sidecar in node pod
	if strings.Contains(pod.Name, "node") {
		var flag bool = false
		for container := range pod.Spec.Containers {
			if pod.Spec.Containers[container].Name == "sdc-monitor" {
				flag = true
			}
		}

		pflxLog.Infof("sdc-monitor container is deployed successfully: %t for %s", flag, pod.Name)
	}

	for container := range pod.Spec.Containers {
		fmt.Println("\t\t", pod.Spec.Containers[container].Name)
		dirName = podDirectoryName + "/" + pod.Spec.Containers[container].Name
		containerDirectoryName := createDirectory(dirName)
		var str string = "Pod status: " + string(pod.Status.Phase)
		filename := pod.Name + ".txt"
		captureLOG(containerDirectoryName, filename, str)
	}
}

// GetLogs accesses the API to get driver/sidecarpod logs of RUNNING pods
func (p PowerFlexStruct) GetLogs(namespace string, optionalFlag string, noOfDays int, driverStorageSystem int) {
	p.namespaceName, _, _ = p.GetDriverDetails(namespace, driverStorageSystem)
	fmt.Println("\n*******************************************************************************")
	var dirName string
	t := time.Now().Format("20060102150405") //YYYYMMDDhhmmss
	dirName = namespace + "_" + t
	namespaceDirectoryName := createDirectory(dirName)
	nodeDirectoryName := ""

	//Capturing describe nodes
	nodes := GetNodes()
	for _, node := range nodes {
		dirName = namespaceDirectoryName + "/" + node
		nodeDirectoryName = createDirectory(dirName)
		p.DescribeNode(node, describe.DescriberSettings{ShowEvents: true}, nodeDirectoryName)
	}
	//Capturing describe pods
	podarray := p.GetPods()
	dateRange := GetDateRange(noOfDays)
	for _, pod := range podarray {
		dirName = namespaceDirectoryName + "/" + pod
		podDirectoryName := createDirectory(dirName)
		p.DescribePods(pod, describe.DescriberSettings{ShowEvents: true}, podDirectoryName)
		if optionalFlag == "True" || optionalFlag == "true" {
			p.DescribePvcs(pod, describe.DescriberSettings{ShowEvents: true}, podDirectoryName)
		}
	}
	p.GetLeaseDetails()

	fmt.Println("\nCollecting Pod Logs (driver logs, sidecar logs)")

	podallns, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		pflxLog.Fatalf("Getting all pods failed with error: %s", err.Error())
	}
	for pod := range podallns.Items {
		if podallns.Items[pod].Namespace != namespace {
			continue
		} else if podallns.Items[pod].Namespace == namespace {
			if podallns.Items[pod].Status.Phase == RunningPodState {
				p.GetRunningPods(namespaceDirectoryName, &podallns.Items[pod], &dateRange, optionalFlag)
			} else {
				p.GetNonRunningPods(namespaceDirectoryName, &podallns.Items[pod])
			}
		}
	}

	// Perform sanitization
	ok := utils.PerformSanitization(clientset, namespace, namespaceDirectoryName)
	if !ok {
		pflxLog.Warnf("Sanitization not performed for %s driver.", namespace)
	}

	errMsg := createTarball(namespaceDirectoryName, ".")

	if errMsg != nil {
		fmt.Printf("Creating tarball %s failed with error: %s", namespaceDirectoryName, errMsg.Error())
		pflxLog.Fatalf("Creating tarball %s failed with error: %s", namespaceDirectoryName, errMsg.Error())
	}
}
