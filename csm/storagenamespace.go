package csm

import (
	"archive/tar"
	"bufio"
	"context"
	utils "csm-logcollector/utils"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	describe "k8s.io/kubectl/pkg/describe"
)

// Logging object
var snsLog = utils.GetLogger()

// StorageNameSpace interface declares log collection methods
type StorageNameSpace interface {
	GetLogs(string, string)
	GetPods() []string
	GetDriverDetails(string) (string, string, string)
	GetLeaseDetails()
	DescribePods(string, describe.DescriberSettings, string)
	ValidateNamespace([]string)
}

// StorageNameSpaceStruct structure declares CSI driver fields
type StorageNameSpaceStruct struct {
	namespaceName string
	drivername    string
	driverversion string
}

var clientset *kubernetes.Clientset
var once sync.Once

// SetClientSetFromConfig creates ClientSet object
func SetClientSetFromConfig() *kubernetes.Clientset {
	once.Do(func() {
		if clientset == nil {
			var kubeconfig *string
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
			} else {
				kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
			}
			flag.Parse()

			config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
			if err != nil {
				snsLog.Errorf("Error while building config object: %s", err.Error())
				panic(err.Error())
			}

			clientset, err = kubernetes.NewForConfig(config)
			if err != nil {
				snsLog.Errorf("Error while building clientset object: %s", err.Error())
				panic(err.Error())
			}
		}
	})
	return clientset
}

// GetClientSetFromConfig returns ClientSet object
func GetClientSetFromConfig() *kubernetes.Clientset {
	return SetClientSetFromConfig()
}

// GetNodes returns the array of nodes in the Kubernetes cluster
func GetNodes() []string {
	// access the API to list Nodes
	// clientset := GetClientSetFromConfig()
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		snsLog.Errorf("Error while getting nodes: %s", err.Error())
		panic(err.Error())
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

// ValidateNamespace validates if given namespace exists in the Kubernetes cluster
func (s StorageNameSpaceStruct) ValidateNamespace(ns []string) {
	fmt.Printf("************ %s\n", s.namespaceName)
	var result bool = false
	for _, x := range ns {
		if x == s.namespaceName {
			result = true
			break
		}
	}

	if result {
		snsLog.Infof("Given Namespace is available in the given environment")
		fmt.Println("Given Namespace is available in the given environment")
	} else {
		snsLog.Errorf("Given Namespace is not available in the given environment.")
		panic("Given Namespace is not available in the given environment.")
	}
}

// GetNamespaces returns the array of namespaces in the Kubernetes cluster
func GetNamespaces() []string {
	// access the API to list Namespaces
	// clientset := GetClientSetFromConfig()
	namespaces, err := clientset.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		snsLog.Errorf("Error while getting namespaces: %s", err.Error())
		panic(err.Error())
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
		snsLog.Errorf("Getting pods in namespace %s failed with error: %s", s.namespaceName, err.Error())
		panic(err.Error())
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
func (p StorageNameSpaceStruct) GetDriverDetails(namespace string) (string, string, string) {
	// Get CSI driver info for a particular namespace
	clientset := GetClientSetFromConfig()
	fmt.Println("\n\nDRIVER INFO..............")
	fmt.Println("=========================")
	podlist, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		snsLog.Errorf("Getting all pods failed with error: %s", err.Error())
		panic(err.Error())
	}
	var driverName string
	var driverVersion string
	for _, pod := range podlist.Items {
		if pod.Namespace == namespace {
			if pod.Status.Phase == "Running" {
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

	p.namespaceName = namespace
	p.drivername = driverName
	p.driverversion = driverVersion
	fmt.Printf("\tNamespace: \t%s\n", p.namespaceName)
	fmt.Printf("\tDriver name: \t%s\n", p.drivername)
	fmt.Printf("\tDriver version: %s\n", p.driverversion)
	snsLog.Debugf("Driver detailes listed: %s, %s, %s", p.namespaceName, p.drivername, p.driverversion)
	return namespace, driverName, driverVersion
}

// GetLeaseDetails gets the lease details
func (s StorageNameSpaceStruct) GetLeaseDetails() {
	// kubectl get leases -n <namespace>
	clientset := GetClientSetFromConfig()
	fmt.Printf("\n\nLease pod for %s..............\n", s.namespaceName)
	fmt.Println("=====================================")
	_ = &coordinationv1.Lease{}
	leasePodList, err := clientset.CoordinationV1().Leases(s.namespaceName).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		snsLog.Errorf("Getting lease details in namespace %s failed with error: %s", s.namespaceName, err.Error())
		panic(err.Error())
	}

	leasepod := "driver-csi-" + s.namespaceName + "-dellemc-com"
	for _, lease := range leasePodList.Items {
		if strings.Contains(lease.Name, leasepod) {
			fmt.Printf("\t%s\n", lease.Name)
			fmt.Printf("\t%s\n", lease.Namespace)
			fmt.Printf("\t%s\n", *lease.Spec.HolderIdentity) // Points to same controller pod for all instances
			snsLog.Debugf("Lease pod detailes: %s, %s, %s", lease.Name, lease.Namespace, *lease.Spec.HolderIdentity)
			fmt.Println()
		}
	}
}

// GetLogs accesses the API to get driver/sidecarpod logs of RUNNING pods
func (s StorageNameSpaceStruct) GetLogs(namespace string, optionalFlag string) {
}

func createDirectory(name string) (dirName string) {
	_, err := os.Stat(name)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(name, 0777)
		if errDir != nil {
			snsLog.Debugf("Error while creating directory: %s", err.Error())
			panic(err.Error())
		}
	}
	return name
}

// DescribePods describes the pods in the given namespace
func (s StorageNameSpaceStruct) DescribePods(podName string, describerSettings describe.DescriberSettings, podDirectoryName string) {
	clientset := GetClientSetFromConfig()
	d := describe.PodDescriber{clientset}
	DescribePodDetails, err := d.Describe(s.namespaceName, podName, describerSettings)
	if err != nil {
		snsLog.Errorf("Describing pod %s in namespace %s failed with error: %s", podName, s.namespaceName, err.Error())
		panic(err.Error())
	}
	filename := podName + "-describe.txt"
	captureLOG(podDirectoryName, filename, DescribePodDetails)
}

func captureLOG(repoName string, filename string, content string) {
	filePath := repoName + "/" + filename
	f, err := os.Create(filePath)
	if err != nil {
		snsLog.Errorf("Creating file %s failed with error: %s", filePath, err.Error())
		panic(err.Error())
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	w.WriteString(content)
	w.Flush()
}

func createTarball(source string, target string) error {
	filename := filepath.Base(source)
	// add the logs.txt file to source directory file
	copy("logs.txt", source)
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
		snsLog.Errorf("Walking the directory %s failed with error: %s", source, errMsgWalk.Error())
		return errMsgWalk
	}

	// remove the logs.txt file to source directory file
	path := source + "/logs.txt"
	errMsgRemove := os.Remove(path)
	if errMsgRemove != nil {
		snsLog.Errorf("Removing file %s failed with error: %s", path, errMsgRemove.Error())
		return errMsgRemove
	}

	fmt.Println("Archive created successfully")
	return nil
}

func copy(src, dst string) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		snsLog.Errorf("Getting info for file %s failed with error: %s", src, err.Error())
		snsLog.Fatal(err.Error())
	}

	if !sourceFileStat.Mode().IsRegular() {
		snsLog.Fatal(err.Error())
	}

	source, err := os.Open(src)
	if err != nil {
		snsLog.Errorf("Opening file %s failed with error: %s", src, err.Error())
		snsLog.Fatal(err.Error())
	}
	defer source.Close()

	dst = dst + "/logs.txt"
	destination, err := os.Create(dst)
	if err != nil {
		snsLog.Errorf("Creating file %s failed with error: %s", dst, err.Error())
		snsLog.Fatal(err.Error())
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	if err != nil {
		snsLog.Errorf("Copying the contents of file failed with error: %s", err.Error())
		snsLog.Fatal(err.Error())
	}
	snsLog.Debugf("logs.txt added to Dir. Copied %d  bytes", nBytes)
}
