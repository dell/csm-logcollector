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
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	utils "csm-logcollector/utils"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"

	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	describe "k8s.io/kubectl/pkg/describe"
)

// constants
const (
	RunningPodState = "Running"
)

// Logging object
var snsLog, logfile = utils.GetLogger()

// StorageNameSpace interface declares log collection methods
type StorageNameSpace interface {
	GetLogs(string, string)
	GetPods() []string
	GetDriverDetails(string) (string, string, string)
	GetLeaseDetails() string
	GetRunningPods(string, *corev1.Pod)
	GetNonRunningPods(string, *corev1.Pod)
	DescribePods(string, describe.DescriberSettings, string)
	DescribePvcs(string, describe.DescriberSettings, string)
}

// StorageNameSpaceStruct structure declares CSI driver fields
type StorageNameSpaceStruct struct {
	namespaceName string
	drivername    string
	driverversion string
}

var once sync.Once
var destinationPath string
var kubeconfigPath string
var clusterIPAddress string
var clusterUsername string
var clusterPassword string
var clientset kubernetes.Interface

// SetClientSetFromConfig creates ClientSet object
func SetClientSetFromConfig() kubernetes.Interface {
	once.Do(func() {
		if clientset == nil {
			var kubeconfig *string
			ReadConfigFile()
			currentIPAddress, err := utils.GetLocalIP()
			if err != nil {
				snsLog.Fatal(err)
			}
			snsLog.Infof("Current node IP: %s", currentIPAddress)

			// verify current system IP
			// container node amd master node are same machine
			if currentIPAddress == clusterIPAddress {
				if kubeconfigPath != "" {
					kubeconfig = flag.String("kubeconfig", kubeconfigPath, "absolute path to the kubeconfig file")
				} else {
					home := homedir.HomeDir()
					kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
				}
				// container node amd master node are different machines
			} else {
				// SCP config file from remote node to container node
				remoteKubeconfigPath := utils.ScpConfigFile(kubeconfigPath, clusterIPAddress, clusterUsername, clusterPassword, ".")
				kubeconfig = flag.String("kubeconfig", remoteKubeconfigPath, "absolute path to the kubeconfig file")
			}
			flag.Parse()

			config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
			if err != nil {
				snsLog.Fatalf("Error while building config object: %s", err.Error())
			}
			clientset, err = kubernetes.NewForConfig(config)
			if err != nil {
				snsLog.Fatalf("Error while building clientset object: %s", err.Error())
			}
		}
	})
	return clientset
}

// GetClientSetFromConfig returns ClientSet object
func GetClientSetFromConfig() kubernetes.Interface {
	return SetClientSetFromConfig()
}

func init() {
	if !strings.Contains(os.Args[0], ".test") {
		clientset = GetClientSetFromConfig()
	}
}

// GetNodes returns the array of nodes in the Kubernetes cluster
func GetNodes() []string {
	// access the API to list Nodes
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		snsLog.Fatalf("Error while getting nodes: %s", err.Error())
	}
	fmt.Println("List of cluster nodes:")
	fmt.Println("=====================")
	length := len(nodes.Items)
	nodearray := make([]string, length)
	for i := 0; i < len(nodes.Items); i++ {
		nodearray[i] = nodes.Items[i].Name
	}
	fmt.Println(nodearray)
	snsLog.Debugf("Cluster nodes listed: %s", nodearray)
	return nodearray
}

// GetNamespaces returns the array of namespaces in the Kubernetes cluster
func GetNamespaces() []string {
	// access the API to list Namespaces

	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		snsLog.Fatalf("Error while getting namespaces: %s", err.Error())
	}
	fmt.Printf("\nThere are %d namespaces in the cluster\n", len(namespaces.Items))
	fmt.Println("List of cluster namespaces:")
	fmt.Println("==========================")

	length := len(namespaces.Items)
	nsarray := make([]string, length)
	for i := 0; i < len(namespaces.Items); i++ {
		nsarray[i] = namespaces.Items[i].Name
	}
	fmt.Println(nsarray)
	snsLog.Debugf("Cluster namespaces listed: %s", nsarray)
	return nsarray
}

// GetPods returns the array of pods in the given namespace
func (s StorageNameSpaceStruct) GetPods() []string {
	// access the API to list Pods of a particular namespace
	clientset := GetClientSetFromConfig()
	fmt.Printf("\n\nList of pods for %s..............\n", s.namespaceName)
	fmt.Println("======================================")
	podList, err := clientset.CoreV1().Pods(s.namespaceName).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		snsLog.Fatalf("Getting pods in namespace %s failed with error: %s", s.namespaceName, err.Error())
	}

	length := len(podList.Items)
	podarray := make([]string, length)
	for i := 0; i < len(podList.Items); i++ {
		podarray[i] = podList.Items[i].Name
	}
	fmt.Println(podarray)
	snsLog.Debugf("Pods in namespace %s listed: %s", s.namespaceName, podarray)
	return podarray
}

// GetDriverDetails populates the CSI driver fields
func (s StorageNameSpaceStruct) GetDriverDetails(namespace string) (string, string, string) {
	// Get CSI driver info for a particular namespace
	fmt.Println("\n\nDRIVER INFO..............")
	fmt.Println("=========================")
	podlist, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		snsLog.Fatalf("Getting all pods failed with error: %s", err.Error())
	}
	var driverName string
	var driverVersion string
	for _, pod := range podlist.Items {
		if pod.Namespace == namespace {
			if pod.Status.Phase == RunningPodState {
				for container := range pod.Spec.Containers {
					if pod.Spec.Containers[container].Name == "driver" {
						driverName = pod.Spec.Containers[container].Name
						image := pod.Spec.Containers[container].Image
						splitString := strings.SplitN(image, ":", 2)
						driverVersion = splitString[1]
					}
				}
			}
		}
	}

	s.namespaceName = namespace
	s.drivername = driverName
	s.driverversion = driverVersion
	fmt.Printf("\tNamespace: \t%s\n", s.namespaceName)
	fmt.Printf("\tDriver name: \t%s\n", s.drivername)
	fmt.Printf("\tDriver version: %s\n", s.driverversion)
	snsLog.Debugf("Driver details listed: %s, %s, %s", s.namespaceName, s.drivername, s.driverversion)
	return namespace, driverName, driverVersion
}

// GetLeaseDetails gets the lease details
func (s StorageNameSpaceStruct) GetLeaseDetails() string {
	// kubectl get leases -n <namespace>
	fmt.Printf("\n\nLease pod for %s..............\n", s.namespaceName)
	fmt.Println("=====================================")
	_ = &coordinationv1.Lease{}
	leasePodList, err := clientset.CoordinationV1().Leases(s.namespaceName).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		snsLog.Fatalf("Getting lease details in namespace %s failed with error: %s", s.namespaceName, err.Error())
	}
	var holder string
	leasepod := "driver-csi-" + s.namespaceName + "-dellemc-com"
	for _, lease := range leasePodList.Items {
		if strings.Contains(lease.Name, leasepod) {
			fmt.Printf("\t%s\n", lease.Name)
			fmt.Printf("\t%s\n", lease.Namespace)
			fmt.Printf("\t%s\n", *lease.Spec.HolderIdentity) // Points to same controller pod for all instances
			snsLog.Debugf("Lease pod details: %s, %s, %s", lease.Name, lease.Namespace, *lease.Spec.HolderIdentity)
			fmt.Println()
			holder = *lease.Spec.HolderIdentity
		}
	}
	return holder
}

// GetLogs accesses the API to get driver/sidecarpod logs of RUNNING pods
func (s StorageNameSpaceStruct) GetLogs(namespace string, optionalFlag string) {
}

func createDirectory(name string) (dirName string) {
	_, err := os.Stat(name)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(name, 0777)
		if errDir != nil {
			snsLog.Fatalf("Error while creating directory: %s", err.Error())
		}
	}
	return name
}

// DescribePods describes the pods in the given namespace
func (s StorageNameSpaceStruct) DescribePods(podName string, describerSettings describe.DescriberSettings, podDirectoryName string) {
	d := describe.PodDescriber{Interface: clientset}
	DescribePodDetails, err := d.Describe(s.namespaceName, podName, describerSettings)
	if err != nil {
		snsLog.Fatalf("Describing pod %s in namespace %s failed with error: %s", podName, s.namespaceName, err.Error())
	}
	filename := podName + "-describe.txt"
	captureLOG(podDirectoryName, filename, DescribePodDetails)
}

// DescribePvcs describes the pvcs in the given namespace
func (s StorageNameSpaceStruct) DescribePvcs(podName string, describerSettings describe.DescriberSettings, podDirectoryName string) {
	podList, err := clientset.CoreV1().Pods(s.namespaceName).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		snsLog.Fatalf("Getting pods in namespace %s failed with error: %s", s.namespaceName, err.Error())
	}

	for i := 0; i < len(podList.Items); i++ {
		var claimName string
		if podList.Items[i].Name == podName {
			for j := 0; j < len(podList.Items[i].Spec.Volumes); j++ {
				pvc := podList.Items[i].Spec.Volumes[j].PersistentVolumeClaim
				if pvc != nil {
					claimName = podList.Items[i].Spec.Volumes[j].PersistentVolumeClaim.ClaimName
					break
				}
			}
			if claimName != "" {
				d := describe.PersistentVolumeClaimDescriber{Interface: clientset}
				DescribePVCDetails, err := d.Describe(s.namespaceName, claimName, describerSettings)
				if err != nil {
					snsLog.Infof("Describing pvc %s in namespace %s failed with error: %s", claimName, s.namespaceName, err.Error())
					return
				}
				filename := claimName + "-describe.txt"
				captureLOG(podDirectoryName, filename, DescribePVCDetails)
			}
		}
	}
}

// GetRunningPods collects log of the running pod in given namespace
func (s StorageNameSpaceStruct) GetRunningPods(namespaceDirectoryName string, pod *corev1.Pod) {
	var dirName string
	fmt.Printf("pod.Name........%s\n", pod.Name)
	fmt.Printf("pod.Status.Phase.......%s\n", pod.Status.Phase)
	dirName = namespaceDirectoryName + "/" + pod.Name
	podDirectoryName := createDirectory(dirName)
	containerCount := len(pod.Spec.Containers)
	fmt.Printf("There are %d containers for the pod\n", containerCount)

	for container := range pod.Spec.Containers {
		fmt.Println("\t", pod.Spec.Containers[container].Name)
		dirName = podDirectoryName + "/" + pod.Spec.Containers[container].Name
		containerDirectoryName := createDirectory(dirName)

		opts := corev1.PodLogOptions{}
		opts.Container = pod.Spec.Containers[container].Name
		req := clientset.CoreV1().Pods(s.namespaceName).GetLogs(pod.Name, &opts)
		podLogs, err := req.Stream(context.TODO())
		if err != nil {
			snsLog.Errorf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
		}
		defer podLogs.Close()
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		if err != nil {
			snsLog.Errorf("Error in copy information from podLogs to buf: %s", err.Error())
		}
		str := buf.String()

		filename := pod.Name + "-" + pod.Spec.Containers[container].Name + ".txt"
		captureLOG(containerDirectoryName, filename, str)
	}
	fmt.Println()
}

// GetNonRunningPods collects log of the nonrunning pod in given namespace
func (s StorageNameSpaceStruct) GetNonRunningPods(namespaceDirectoryName string, pod *corev1.Pod) {
	var dirName string
	fmt.Printf("pod.Name........%s\n", pod.Name)
	fmt.Printf("pod.Status.Phase.......%s\n", pod.Status.Phase)
	containerCount := len(pod.Spec.Containers)
	fmt.Printf("There are %d containers for the pod\n", containerCount)
	dirName = namespaceDirectoryName + "/" + pod.Name
	podDirectoryName := createDirectory(dirName)

	for container := range pod.Spec.Containers {
		fmt.Println("\t", pod.Spec.Containers[container].Name)
		dirName = podDirectoryName + "/" + pod.Spec.Containers[container].Name
		containerDirectoryName := createDirectory(dirName)
		var str string = "Pod status: not running"
		filename := pod.Name + ".txt"
		captureLOG(containerDirectoryName, filename, str)
		fmt.Println()
	}
}

func captureLOG(repoName string, filename string, content string) {
	filePath := repoName + "/" + filename
	f, err := os.Create(filePath)
	if err != nil {
		snsLog.Fatalf("Creating file %s failed with error: %s", filePath, err.Error())
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	w.WriteString(content)
	w.Flush()
}

// ReadConfigFile reads the application configuration file
func ReadConfigFile() {
	_, err := os.Stat("config.yml")
	if err == nil {
		yamlFile, err := ioutil.ReadFile("config.yml")
		if err != nil {
			snsLog.Fatalf("Reading configuration file failed with error %v ", err)
		}

		data := make(map[interface{}]interface{})
		err = yaml.Unmarshal(yamlFile, data)
		if err != nil {
			snsLog.Fatalf("Unmarshalling configuration file failed with error %v", err)
		}

		for k, v := range data {
			if k == "destination_path" {
				destinationPath = fmt.Sprintf("%s", v)
				snsLog.Infof("destination path: %s", destinationPath)
			}

			if k == "kubeconfig_details" {
				// To access kubeconfig_details, assert type of data["kubeconfig_details"] to map[interface{}]interface{}
				kubeconfigDetails, ok := data["kubeconfig_details"].(map[interface{}]interface{})
				if !ok {
					snsLog.Fatalf("kubeconfig_details is not a map!")
				}

				for key, value := range kubeconfigDetails {
					// type assertion from interface{} type to string type
					key, ok1 := key.(string)
					value, ok2 := value.(string)
					if !ok1 || !ok2 {
						snsLog.Fatalf("key/value is not string!")
					}
					if len(strings.TrimSpace(value)) != 0 {
						if key == "path" {
							kubeconfigPath = value
						}
						if key == "ip_address" {
							clusterIPAddress = value
						}
						if key == "username" {
							clusterUsername = value
						}
						if key == "password" {
							clusterPassword = value
						}
					} else {
						snsLog.Infof("No value found for kubeconfig_details sub-key: %s", key)
					}
				}
			}
		}
	}
}

func createTarball(source string, target string) error {
	filename := filepath.Base(source)
	// add the log file file to source directory
	copy(logfile, source)
	target = filepath.Join(target, fmt.Sprintf("%s.tar.gz", filename))
	tarfile, err := os.Create(target)
	if err != nil {
		snsLog.Errorf("Creating file %s failed with error: %s", target, err.Error())
		return err
	}
	defer tarfile.Close()

	tarball := tar.NewWriter(tarfile)
	defer tarball.Close()

	info, err := os.Stat(source)
	if err != nil {
		snsLog.Errorf("Getting info for file %s failed with error: %s", source, err.Error())
		return err
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	errMsgWalk := filepath.Walk(source,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				snsLog.Errorf("Creating fileinfo object failed with error: %s", err.Error())
				return err
			}
			header, err := tar.FileInfoHeader(info, info.Name())
			if err != nil {
				snsLog.Errorf("Creating header object failed with error: %s", err.Error())
				return err
			}

			if baseDir != "" {
				header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
			}

			if err := tarball.WriteHeader(header); err != nil {
				snsLog.Errorf("Writing header object failed with error: %s", err.Error())
				return err
			}

			if info.IsDir() {
				return nil
			}

			file, err := os.Open(path)
			if err != nil {
				snsLog.Errorf("Opening file %s failed with error: %s", path, err.Error())
				return err
			}
			defer file.Close()
			_, err = io.Copy(tarball, file)
			return err
		})
	if errMsgWalk != nil {
		snsLog.Errorf("Navigating through the directory %s failed with error: %s", source, errMsgWalk.Error())
		return errMsgWalk
	}

	// remove the log file from source directory
	path := source + "/" + logfile
	errMsgRemove := os.Remove(path)
	if errMsgRemove != nil {
		snsLog.Errorf("Removing file %s failed with error: %s", path, errMsgRemove.Error())
		return errMsgRemove
	}

	// Move tarball to given path if provided
	if destinationPath != "" {
		if strings.HasSuffix(destinationPath, "/") {
			destinationPath = destinationPath + target
		} else {
			destinationPath = destinationPath + "/" + target
		}

		err := os.Rename(target, destinationPath)
		if err != nil {
			snsLog.Errorf("Moving file %s failed with error: %s", target, err.Error())
			return err
		}
	}

	fmt.Println("Archive created successfully")

	// cleanup call
	cleanup()
	return nil
}

func copy(src, dst string) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		snsLog.Fatalf("Getting info for file %s failed with error: %s", src, err.Error())
	}

	if !sourceFileStat.Mode().IsRegular() {
		snsLog.Fatal(err.Error())
	}

	source, err := os.Open(src)
	if err != nil {
		snsLog.Fatalf("Opening file %s failed with error: %s", src, err.Error())
	}
	defer source.Close()

	dst = dst + "/" + logfile
	destination, err := os.Create(dst)
	if err != nil {
		snsLog.Fatalf("Creating file %s failed with error: %s", dst, err.Error())
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	if err != nil {
		snsLog.Fatalf("Copying the contents of file failed with error: %s", err.Error())
	}
	snsLog.Debugf("log file added to Dir. Copied %d  bytes", nBytes)
}

func cleanup() {
	_, err1 := os.Stat("config")
	_, err2 := os.Stat("RemoteClusterSecretFiles")

	if err1 == nil || err2 == nil {
		snsLog.Infof("Cleanup started.")
		e1 := os.Remove("config")
		if e1 != nil {
			snsLog.Infof("Error: %s", e1)
		}
		e2 := os.RemoveAll("RemoteClusterSecretFiles")
		if e2 != nil {
			snsLog.Infof("Error: %s", e2)
		}
		snsLog.Infof("Cleanup completed.")
	}
}
