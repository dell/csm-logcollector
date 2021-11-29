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
	"strings"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
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
func (p PowerStoreStruct) GetLeaseDetails() {
	fmt.Printf("\n\nLease pod for %s..............\n", p.namespaceName)
	fmt.Println("=====================================")
	_ = &coordinationv1.Lease{}
	leasePodList, err := clientset.CoordinationV1().Leases(p.namespaceName).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		psLog.Fatalf("Getting lease details in namespace %s failed with error: %s", p.namespaceName, err.Error())
	}
	leasepod := "external-attacher-leader-" + p.namespaceName + "-dellemc-com"

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

// GetLogs accesses the API to get driver/sidecarpod logs of RUNNING pods
func (p PowerStoreStruct) GetLogs(namespace string, optionalFlag string) {
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

	for _, pod := range podarray {
		dirName = namespaceDirectoryName + "/" + pod
		podDirectoryName := createDirectory(dirName)
		p.DescribePods(pod, describe.DescriberSettings{ShowEvents: true}, podDirectoryName)
	}

	p.GetLeaseDetails()
	// access the API to get driver/sidecarpod logs of RUNNING pods

	fmt.Printf("Optional flag: %s\n", optionalFlag)
	fmt.Println("\n\nCollecting RUNNING POD LOGS (driver logs, sidecar logs)..........")

	podallns, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		psLog.Fatalf("Getting all pods failed with error: %s", err.Error())
	}
	for _, pod := range podallns.Items {
		if pod.Namespace != namespace {
			continue
		} else if pod.Namespace == namespace {
			if pod.Status.Phase == RunningPodState {
				p.GetRunningPods(namespaceDirectoryName, &pod)
			} else {
				p.GetNonRunningPods(namespaceDirectoryName, &pod)
			}
		}
	}

	errMsg := createTarball(namespaceDirectoryName, ".")

	if errMsg != nil {
		psLog.Fatalf("Creating tarball %s failed with error: %s", namespaceDirectoryName, errMsg.Error())
	}
}