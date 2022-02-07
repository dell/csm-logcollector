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
	"context"
	utils "csm-logcollector/utils"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	describe "k8s.io/kubectl/pkg/describe"
)

// Logging object
var pmaxLog, _ = utils.GetLogger()

// PowerMaxStruct for PowerMax platform
type PowerMaxStruct struct {
	StorageNameSpaceStruct
}

// LeaseHolder variable to old the lease holder
var LeaseHolder string

// GetRunningPods is overridden for PowerMax specific implementation
func (p PowerMaxStruct) GetRunningPods(namespaceDirectoryName string, pod *corev1.Pod) {
	var dirName string
	fmt.Printf("pod.Name........%s\n", pod.Name)
	fmt.Printf("pod.Status.Phase.......%s\n", pod.Status.Phase)
	dirName = namespaceDirectoryName + "/" + pod.Name
	podDirectoryName := createDirectory(dirName)

	// check for reverse-proxy sidecar in controller pod
	if pod.Name == LeaseHolder {
		var flag bool = false
		for container := range pod.Spec.Containers {
			if pod.Spec.Containers[container].Name == "reverseproxy" {
				flag = true
				break
			}
		}
		pmaxLog.Infof("Reverse Proxy is deployed as sidecar: %t", flag)
	}

	str := "Pod " + pod.Name + " is in running state\n"
	filename := pod.Name + ".txt"
	captureLOG(podDirectoryName, filename, str)
}

// GetNonRunningPods is overridden for PowerMax specific implementation
func (p PowerMaxStruct) GetNonRunningPods(namespaceDirectoryName string, pod *corev1.Pod, daterange *metav1.Time) {
	var dirName string
	fmt.Printf("pod.Name........%s\n", pod.Name)
	fmt.Printf("pod.Status.Phase.......%s\n", pod.Status.Phase)
	dirName = namespaceDirectoryName + "/" + pod.Name
	podDirectoryName := createDirectory(dirName)
	containerCount := len(pod.Spec.Containers)
	fmt.Printf("There are %d containers for this pod\n", containerCount)

	if daterange != nil {
		podLogOpts := corev1.PodLogOptions{}
		podLogOpts.SinceTime = daterange
		fmt.Printf("Time: %v", podLogOpts.SinceTime)
		podLogs := clientset.CoreV1().Pods("").GetLogs(pod.Name, &podLogOpts)
		fmt.Printf("Logs: %v", podLogs)
	}

	// check for reverse-proxy sidecar in controller pod
	if pod.Name == LeaseHolder {
		var flag bool = false
		for container := range pod.Spec.Containers {
			if pod.Spec.Containers[container].Name == "reverseproxy" {
				flag = true
				break
			}
		}

		pmaxLog.Infof("Reverse Proxy is deployed as sidecar: %t", flag)
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
func (p PowerMaxStruct) GetLogs(namespace string, optionalFlag string, noofdays int) {
	p.namespaceName, _, _ = p.GetDriverDetails(namespace)
	fmt.Println("\n*******************************************************************************")
	GetNodes()
	podarray := p.GetPods()
	daterange := GetDateRange(noofdays)
	var dirName string
	t := time.Now().Format("20060102150405") //YYYYMMDDhhmmss
	dirName = namespace + "_" + t
	namespaceDirectoryName := createDirectory(dirName)

	for _, pod := range podarray {
		dirName = namespaceDirectoryName + "/" + pod
		podDirectoryName := createDirectory(dirName)
		p.DescribePods(pod, describe.DescriberSettings{ShowEvents: true}, podDirectoryName)
		if optionalFlag == "True" || optionalFlag == "true" {
			p.DescribePvcs(pod, describe.DescriberSettings{ShowEvents: true}, podDirectoryName)
		}
	}

	LeaseHolder = p.GetLeaseDetails()
	fmt.Println("\n*******************************************************************************")

	fmt.Printf("\nOptional flag: %s", optionalFlag)
	fmt.Println("\nCollecting Running Pod Logs (driver logs, sidecar logs)")

	podallns, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		pmaxLog.Fatalf("Getting all pods failed with error: %s", err.Error())
	}
	for pod := range podallns.Items {
		if podallns.Items[pod].Namespace != namespace {
			continue
		} else if podallns.Items[pod].Namespace == namespace {
			if podallns.Items[pod].Status.Phase == RunningPodState {
				p.GetRunningPods(namespaceDirectoryName, &podallns.Items[pod])
				fmt.Println("\t*************************************************************")
				pmaxLog.Infof("Logs collected for runningpods of %s", namespace)
			} else {
				p.GetNonRunningPods(namespaceDirectoryName, &podallns.Items[pod], &daterange)
				fmt.Println("\t*************************************************************")
				pmaxLog.Infof("Logs collected for non-runningpods of %s", namespace)
			}
		}
	}

	// Perform sanitization
	ok := utils.PerformSanitization(namespaceDirectoryName)
	if !ok {
		pmaxLog.Warnf("Sanitization not performed for %s driver.", namespace)
	}

	errMsg := createTarball(namespaceDirectoryName, ".")

	if errMsg != nil {
		pmaxLog.Fatalf("Creating tarball %s failed with error: %s", namespaceDirectoryName, errMsg.Error())
	}
}
