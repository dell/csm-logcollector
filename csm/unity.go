package csm

import (
	"fmt"
	 "context"
	 "bytes"
	 "io"
	 "time"
	 corev1 "k8s.io/api/core/v1"
	 metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	 describe "k8s.io/kubectl/pkg/describe"
	 utils "csm-logcollector/utils"
)

// Logging object
var unityLog = utils.GetLogger()

type UnityStruct struct {
	StorageNameSpaceStruct
}

func runningpods_unity(namespaceDirectoryName string, pod *corev1.Pod) {
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
				unityLog.Errorf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
				fmt.Printf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
			}
			defer podLogs.Close()
			buf := new(bytes.Buffer)
			_, err = io.Copy(buf, podLogs)
			if err != nil {
				unityLog.Errorf("Error in copy information from podLogs to buf: %s", err.Error())
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
			unityLog.Errorf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
			fmt.Printf("Opening stream for pod %s in namespace %s failed with error: %s", pod.Name, pod.Namespace, err.Error())
		}
		defer podLogs.Close()
		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		if err != nil {
			unityLog.Errorf("Error in copy information from podLogs to buf: %s", err.Error())
			fmt.Printf("Error in copy information from podLogs to buf: %s", err.Error())
		}
		str := buf.String()
		
		filename := pod.Name + ".txt"
		captureLOG(containerDirectoryName, filename, str)
		fmt.Println()
	}
}

func nonrunningpods_unity(namespaceDirectoryName string, pod *corev1.Pod) {
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
			var str string
			str = "Pod status: not running"
			filename := pod.Name + ".txt"
			captureLOG(containerDirectoryName, filename, str)
			fmt.Println()
		}
	} else {
		dirName = podDirectoryName + "/" + pod.Spec.Containers[0].Name
		containerDirectoryName := createDirectory(dirName)
		var str string
		str = "Pod status: not running"
		
		filename := pod.Name + ".txt"
		captureLOG(containerDirectoryName, filename, str)
		fmt.Println()
	}
}

func (p UnityStruct ) GetLogs(namespace string, optional_flag string) {
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
	fmt.Printf("Optional flag: %s", optional_flag)
	fmt.Println("\n\nCollecting RUNNING POD LOGS (driver logs, sidecar logs)..........")

	pod__, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		unityLog.Errorf("Getting all pods failed with error: %s", err.Error())
		panic(err.Error())
	}
	for _, pod := range pod__.Items {
		if pod.Namespace == namespace {
			if pod.Status.Phase == "Running" {
				runningpods_unity(namespaceDirectoryName, &pod)
			} else {
				nonrunningpods_unity(namespaceDirectoryName, &pod)
			}
		}
	}

	errMsg := createTarball(namespaceDirectoryName, ".")

    if errMsg != nil {
		unityLog.Errorf("Creating tarball %s failed with error: %s", namespaceDirectoryName, errMsg.Error())
        panic(errMsg.Error())
    }
}

