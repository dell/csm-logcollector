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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	describe "k8s.io/kubectl/pkg/describe"
)

// Logging object
var unityLog, _ = utils.GetLogger()

// UnityStruct for Unity platform
type UnityStruct struct {
	StorageNameSpaceStruct
}

// GetLogs accesses the API to get driver/sidecarpod logs of RUNNING pods
func (p UnityStruct) GetLogs(namespace string, optionalFlag string, noOfDays int, driverStorageSystem int) {
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
	// access the API to get driver/sidecarpod logs of RUNNING pods
	fmt.Printf("Optional flag: %s", optionalFlag)
	fmt.Println("\n\nCollecting RUNNING POD LOGS (driver logs, sidecar logs)..........")

	podallns, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		unityLog.Fatalf("Getting all pods failed with error: %s", err.Error())
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
		unityLog.Warnf("Sanitization not performed for %s driver.", namespace)
	}

	errMsg := createTarball(namespaceDirectoryName, ".")

	if errMsg != nil {
		fmt.Printf("Creating tarball %s failed with error: %s\n", namespaceDirectoryName, errMsg.Error())
		unityLog.Fatalf("Creating tarball %s failed with error: %s", namespaceDirectoryName, errMsg.Error())
	}
}
