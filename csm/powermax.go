package csm

import (
	"fmt"
	 "bytes"
	 "context"
	 "time"
	 "io"
	 corev1 "k8s.io/api/core/v1"
	 metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	 describe "k8s.io/kubectl/pkg/describe"
	 utils "csm-logcollector/utils"
)

// Logging object
var pmaxLog = utils.GetLogger()

type PowerMaxStruct struct {
	StorageNameSpaceStruct
}

func runningpods_powermax(namespaceDirectoryName string, pod *corev1.Pod) {
	var dirName string
	fmt.Printf("\tpod.Name........%s\n", pod.Name)
	dirName = namespaceDirectoryName + "/" + pod.Name
	podDirectoryName := createDirectory(dirName)
	fmt.Printf("\tThere are %d containers for this pod\n", len(pod.Spec.Containers))
	if len(pod.Spec.Containers) > 1 {
		for container := range pod.Spec.Containers {
			fmt.Println("\t\t", pod.Spec.Containers[container].Name)
			dirName = podDirectoryName + "/" + pod.Spec.Containers[container].Name
			containerDirectoryName := createDirectory(dirName)

			opts := corev1.PodLogOptions{}
			opts.Container = pod.Spec.Containers[container].Name
			req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &opts)
			podLogs, err := req.Stream(context.TODO())
			if err != nil {
				pmaxLog.Errorf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
				fmt.Printf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
			}
			defer podLogs.Close()
			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, podLogs)
			if err != nil {
				pmaxLog.Errorf("Error in copy information from podLogs to buf: %s", err.Error())
				fmt.Printf("Error in copy information from podLogs to buf: %s", err.Error())
			}
			str := buf.String()
			filename := pod.Name + "-" + pod.Spec.Containers[container].Name + ".txt"
			fmt.Println("\tLOG collected in file..........\n")
			captureLOG(containerDirectoryName, filename, str)
		}
	} else {
		dirName = podDirectoryName + "/" + pod.Spec.Containers[0].Name
		containerDirectoryName := createDirectory(dirName)

		req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{})
		podLogs, err := req.Stream(context.TODO())
		if err != nil {
			pmaxLog.Errorf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
			fmt.Printf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
		}
		defer podLogs.Close()
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		if err != nil {
			pmaxLog.Errorf("Error in copy information from podLogs to buf: %s", err.Error())
			fmt.Printf("Error in copy information from podLogs to buf: %s", err.Error())
		}
		str := buf.String()
		filename := pod.Name + ".txt"
		fmt.Println("LOG collected in file..........\n")
		captureLOG(containerDirectoryName, filename, str)
	}
}

func nonrunningpods_powermax(namespaceDirectoryName string, pod *corev1.Pod) {
	var dirName string
	fmt.Printf("\tpod.Name........%s\n", pod.Name)
	dirName = namespaceDirectoryName + "/" + pod.Name
	podDirectoryName := createDirectory(dirName)
	fmt.Printf("\tThere are %d containers for this pod\n", len(pod.Spec.Containers))
	if len(pod.Spec.Containers) > 1 {
		for container := range pod.Spec.Containers {
			fmt.Println("\t\t", pod.Spec.Containers[container].Name)
			dirName = podDirectoryName + "/" + pod.Spec.Containers[container].Name
			containerDirectoryName := createDirectory(dirName)
			var str string
			str = "Pod status: " + string(pod.Status.Phase)
			filename := pod.Name + ".txt"
			fmt.Println("\tLOG collected in file..........\n")
			captureLOG(containerDirectoryName, filename, str)
		}
	} else {
		dirName = podDirectoryName + "/" + pod.Spec.Containers[0].Name
		containerDirectoryName := createDirectory(dirName)
		var str string
		str = "Pod status: " + string(pod.Status.Phase)
		filename := pod.Name + ".txt"
		fmt.Println("LOG collected in file..........\n")
		captureLOG(containerDirectoryName, filename, str)
	}
}

// access the API to get driver/sidecarpod logs of RUNNING pods
func (p PowerMaxStruct ) GetLogs(namespace string, optional_flag string) {
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
	fmt.Println("\n*******************************************************************************")	

	fmt.Printf("\nOptional flag: %s", optional_flag)
	fmt.Println("\nCollecting Running Pod Logs (driver logs, sidecar logs)")

	pod__, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		pmaxLog.Errorf("Getting all pods failed with error: %s", err.Error())
		panic(err.Error())
	}
	for _, pod := range pod__.Items {
		if pod.Namespace == namespace {
			if pod.Status.Phase == "Running" {
				runningpods_powermax(namespaceDirectoryName, &pod)
				fmt.Println("\t*************************************************************")
				pmaxLog.Infof("Logs collected for runningpods of %s", namespace)			
			} else {
				nonrunningpods_powermax(namespaceDirectoryName, &pod)
				fmt.Println("\t*************************************************************")
				pmaxLog.Infof("Logs collected for non-runningpods of %s", namespace)
			}
		}
	}
	errMsg := createTarball(namespaceDirectoryName, ".")

    if errMsg != nil {
		pmaxLog.Errorf("Creating tarball %s failed with error: %s", namespaceDirectoryName, errMsg.Error())
        panic(errMsg.Error())
    }
}
