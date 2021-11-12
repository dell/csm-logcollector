package csm

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"archive/tar"
	"bufio"
	"os"
	"io"
	"strings"
	"sync"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	describe "k8s.io/kubectl/pkg/describe"
	coordinationv1 "k8s.io/api/coordination/v1"
	utils "csm-logcollector/utils"
)

// Logging object
var snsLog = utils.GetLogger()

type StorageNameSpace interface {
	GetNodes() []string
	GetNamespaces() []string
	GetLogs(string, string)
	GetPods(string) []string
	GetDriverDetails(string)
	GetLeaseDetails(string)
	DescribePods(string, string, describe.DescriberSettings, string)
	ValidateNamespace([] string, string)
}

type StorageNameSpaceStruct struct {
	namespace_name string
	drivername string
	driverversion string
}

var clientset *kubernetes.Clientset
var once sync.Once

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

func GetClientSetFromConfig() *kubernetes.Clientset {
	return SetClientSetFromConfig()
}

func (_ StorageNameSpaceStruct ) GetNodes() []string{
	// access the API to list Nodes
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		snsLog.Errorf("Error while getting nodes: %s", err.Error())
		panic(err.Error())
	}
	fmt.Println("List of cluster nodes:")
	fmt.Println("=====================")
	length := len(nodes.Items)
	nodearray := make([]string, length)
	for i:=0; i < len(nodes.Items); i++ {
		nodearray[i] = nodes.Items[i].Name
    }
	fmt.Println(nodearray)
	snsLog.Debugf("Cluster nodes listed: %s", nodearray)
	return nodearray
}

func (_ StorageNameSpaceStruct ) ValidateNamespace(ns []string, namespace string){
	var result bool = false
    for _, x := range ns {
        if x == namespace {
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

func (_ StorageNameSpaceStruct ) GetNamespaces() []string{

	// access the API to list Namespaces
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
	for i:=0; i < len(namespaces.Items); i++ {
		nsarray[i] = namespaces.Items[i].Name
    }
	fmt.Println(nsarray)
	snsLog.Debugf("Cluster namespaces listed: %s", nsarray)
	return nsarray
}

func (_ StorageNameSpaceStruct ) GetPods(namespace string) []string{
	// access the API to list Pods of a particular namespace
	fmt.Printf("\n\nList of pods for %s..............\n", namespace)
	fmt.Println("======================================")
	podList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		snsLog.Errorf("Getting pods in namespace %s failed with error: %s", namespace, err.Error())
		panic(err.Error())
	}
	
	length := len(podList.Items)
	podarray := make([]string, length)
	for i:=0; i < len(podList.Items); i++ {
		podarray[i] = podList.Items[i].Name
    }
	fmt.Println(podarray)
    snsLog.Debugf("Pods in namespace %s listed: %s", namespace, podarray)
    return podarray
}

func (p StorageNameSpaceStruct) GetDriverDetails(namespace string) {
	// Get CSI driver info for a particular namespace
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

	p.namespace_name = namespace
	p.drivername = driverName
	p.driverversion = driverVersion
	fmt.Printf("\tNamespace: \t%s\n", p.namespace_name)
	fmt.Printf("\tDriver name: \t%s\n", p.drivername)
	fmt.Printf("\tDriver version: %s\n", p.driverversion)
	snsLog.Debugf("Driver detailes listed: %s, %s, %s", p.namespace_name, p.drivername, p.driverversion)
}

func (_ StorageNameSpaceStruct ) GetLeaseDetails(namespace string) {
	// kubectl get leases -n <namespace>
    fmt.Printf("\n\nLease pod for %s..............\n", namespace)
    fmt.Println("=====================================\n")
    _ = &coordinationv1.Lease{}
    leasePodList, err := clientset.CoordinationV1().Leases(namespace).List(context.TODO(), metav1.ListOptions{})
    if err != nil {
		snsLog.Errorf("Getting lease details in namespace %s failed with error: %s", namespace, err.Error())
        panic(err.Error())
    }
	
	leasepod := "driver-csi-" + namespace + "-dellemc-com"
    for _, lease := range leasePodList.Items {
		if strings.Contains(lease.Name, leasepod) {
        	fmt.Printf("\t%s\n", lease.Name)
        	fmt.Printf("\t%s\n", lease.Namespace)
        	fmt.Printf("\t%s\n", *lease.Spec.HolderIdentity)      // Points to same controller pod for all instances
			snsLog.Debugf("Lease pod detailes: %s, %s, %s", lease.Name, lease.Namespace, *lease.Spec.HolderIdentity)
        	fmt.Println()
		}
    }
}

func (_ StorageNameSpaceStruct ) GetLogs(namespace string, optional_flag string) {
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

func (_ StorageNameSpaceStruct ) DescribePods(namespace string, podName string, describerSettings describe.DescriberSettings, podDirectoryName string) {
	d := describe.PodDescriber{clientset}
	DescribePodDetails, err := d.Describe(namespace, podName, describerSettings)
	if err != nil {
		snsLog.Errorf("Describing pod %s in namespace %s failed with error: %s", podName, namespace, err.Error())
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
		snsLog.Errorf("Copying the contents of file %s to file %s failed with error: %s", source, destination, err.Error())
		snsLog.Fatal(err.Error())
	}
	snsLog.Debugf("logs.txt added to Dir. Copied ", nBytes, " bytes")
}