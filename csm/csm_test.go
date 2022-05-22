package csm

import (
	"context"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	utils "csm-logcollector/utils"

	"github.com/google/go-cmp/cmp"
	coordinationv1 "k8s.io/api/coordination/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	describe "k8s.io/kubectl/pkg/describe"
)

func CreatePod(clientset kubernetes.Interface, namespace string, name string, containerName string) *v1.Pod {
	pod := &v1.Pod{ObjectMeta: meta_v1.ObjectMeta{Name: name, Namespace: namespace}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: containerName}}}}
	resp, _ := clientset.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, meta_v1.CreateOptions{})
	return resp
}

func CreatePVC(clientset kubernetes.Interface, namespace string, name string) *v1.PersistentVolumeClaim {
	pvc := &v1.PersistentVolumeClaim{ObjectMeta: meta_v1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: v1.PersistentVolumeClaimSpec{AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.ResourceRequirements{Requests: v1.ResourceList{v1.ResourceName(v1.ResourceStorage): resource.MustParse("5Gi")}}}}
	resp, _ := clientset.CoreV1().PersistentVolumeClaims(namespace).Create(context.TODO(), pvc, meta_v1.CreateOptions{})
	return resp
}

func CreatePodPVC(clientset kubernetes.Interface, namespace string, name string, containerName string, pvcName string) *v1.Pod {
	pod := &v1.Pod{ObjectMeta: meta_v1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: v1.PodSpec{Containers: []v1.Container{{Name: containerName}},
			Volumes: []v1.Volume{{Name: "pv-storage",
				VolumeSource: v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName}}}}}}
	resp, _ := clientset.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, meta_v1.CreateOptions{})
	return resp
}

func CreateNamespace(clientset kubernetes.Interface, namespaceName string) *v1.Namespace {
	namespace := &v1.Namespace{ObjectMeta: meta_v1.ObjectMeta{Name: namespaceName}}
	resp, _ := clientset.CoreV1().Namespaces().Create(context.TODO(), namespace, meta_v1.CreateOptions{})
	return resp
}

func CreateLease(clientset kubernetes.Interface, name string, namespaceName string, hid string) *coordinationv1.Lease {
	lease := &coordinationv1.Lease{ObjectMeta: meta_v1.ObjectMeta{Name: name, Namespace: namespaceName}, Spec: coordinationv1.LeaseSpec{HolderIdentity: &hid}}
	resp, _ := clientset.CoordinationV1().Leases(namespaceName).Create(context.TODO(), lease, meta_v1.CreateOptions{})
	return resp
}

func pod(namespace, podName string, image string, driverName string) *v1.Pod {
	return &v1.Pod{ObjectMeta: meta_v1.ObjectMeta{Namespace: namespace, Name: podName}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: driverName, Image: image}}}}
}

func CreateNodes(client kubernetes.Interface, name string) *v1.Node {
	node := &v1.Node{ObjectMeta: meta_v1.ObjectMeta{Name: name}}
	resp, _ := client.CoreV1().Nodes().Create(context.TODO(), node, meta_v1.CreateOptions{})
	return resp
}

func TestGetPods(t *testing.T) {
	var st StorageNameSpaceStruct
	st.namespaceName = "unity"
	var tests = []struct {
		description string
		namespace   string
		expected    []string
		objs        []runtime.Object
	}{
		{"pods list for a namespace",
			st.namespaceName,
			[]string{"pod_1", "pod_2"},
			[]runtime.Object{pod(st.namespaceName, "pod_1", "driver_image_1:v1", "driver_1"),
				pod(st.namespaceName, "pod_2", "driver_image_2:v2", "driver_2")}},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset(test.objs...)
			actual := st.GetPods()
			if diff := cmp.Diff(actual, test.expected); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expected, diff)
				return
			}
		})
	}
}

func TestGetDriverDetails(t *testing.T) {
	var st StorageNameSpaceStruct
	var tests = []struct {
		description       string
		expectedNamespace string
		objs              []runtime.Object
	}{
		{"driver details",
			"csi-powerstore",
			[]runtime.Object{pod("csi-powerstore", "test_pod_1", "driver", "driver_image_1:v1.0.0")}},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset(test.objs...)
			gotNamespace, _, _ := st.GetDriverDetails("csi-powerstore", 3)
			if diff := cmp.Diff(gotNamespace, test.expectedNamespace); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedNamespace, diff)
				return
			}
		})
	}
}

func TestGetDescribePvcs(t *testing.T) {
	var st StorageNameSpaceStruct
	st.namespaceName = "csi-powerstore"
	var tests = []struct {
		description  string
		expectedData string
	}{
		{"Describe pvc",
			"Name:          iscsipvc",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			CreatePVC(clientset, "csi-powerstore", "iscsipvc")
			CreatePodPVC(clientset, "csi-powerstore", "test-describe-pvc", "attacher", "iscsipvc")
			st.DescribePvcs("test-describe-pvc", describe.DescriberSettings{ShowEvents: true}, createDirectory("csi-powerstore/test-describe-pvc"))
			file := "csi-powerstore/test-describe-pvc/iscsipvc-describe.txt"
			data, _ := ioutil.ReadFile(file)
			got := string(data)
			strings.SplitN(got, "\n", 1)
			if !strings.Contains(got, test.expectedData) {
				t.Errorf("%T differ (-got, +want): \n\t\t - %s\n\t\t + %s", test.expectedData, got, test.expectedData)
				return
			}
		})
	}
}
func TestGetDescribePods(t *testing.T) {
	var st StorageNameSpaceStruct
	st.namespaceName = "vxflexos-namespace"
	var tests = []struct {
		description  string
		expectedData string
	}{
		{"Describe pod",
			"Name:         test-describe-pod",
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			CreatePod(clientset, "vxflexos-namespace", "test-describe-pod", "sdc-monitor")
			st.DescribePods("test-describe-pod", describe.DescriberSettings{ShowEvents: true}, createDirectory("vxflexos-namespace/test-describe-pod"))
			file := "vxflexos-namespace/test-describe-pod/test-describe-pod-describe.txt"
			data, _ := ioutil.ReadFile(file)
			got := string(data)
			strings.SplitN(got, "\n", 1)
			if !strings.Contains(got, test.expectedData) {
				t.Errorf("%T differ (-got, +want): \n\t\t - %s\n\t\t + %s", test.expectedData, got, test.expectedData)
				return
			}
		})
	}
}

func TestGetRunningPods(t *testing.T) {
	type tests = []struct {
		description string
		namespace   string
		expected    string
	}

	// Unity, PowerScale, PowerStore
	var st StorageNameSpaceStruct
	var stTests = tests{
		{"get running pod logs", "correct-namespace", "Pod test-running-pod is in running state\n"},
	}
	for _, test := range stTests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			namespaceDirectoryName := "common-pod-logs"
			dateRange := meta_v1.Now()
			optionalFlag := "false"
			pod := CreatePod(clientset, "correct-namespace", "test-running-pod", "test-container")
			st.GetRunningPods(namespaceDirectoryName, pod, &dateRange, optionalFlag)
			file := "common-pod-logs/test-running-pod/test-running-pod.txt"
			data, _ := ioutil.ReadFile(file)
			got := string(data)
			if diff := cmp.Diff(got, test.expected); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expected, diff)
				return
			}
		})
	}

	// PowerFlex
	var pflx PowerFlexStruct
	var pflxTests = tests{
		{"get running pod logs", "vxflexos-namespace", "Pod test-running-pod is in running state\n"},
	}

	for _, test := range pflxTests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			namespaceDirectoryName := "vxflexos-pod-logs"
			pod := CreatePod(clientset, "vxflexos-namespace", "test-running-pod", "sdc-monitor")
			dateRange := meta_v1.Now()
			optionalFlag := "false"
			pflx.GetRunningPods(namespaceDirectoryName, pod, &dateRange, optionalFlag)
			file := "vxflexos-pod-logs/test-running-pod/test-running-pod.txt"
			data, _ := ioutil.ReadFile(file)
			got := string(data)
			if diff := cmp.Diff(got, test.expected); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expected, diff)
				return
			}
		})
	}

	// PowerMax
	var pmx PowerMaxStruct
	var pmxTests = tests{
		{"get running pod logs", "powermax-namespace", "Pod test-running-pod is in running state\n"},
	}

	for _, test := range pmxTests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			namespaceDirectoryName := "powermax-pod-logs"
			pod := CreatePod(clientset, "powermax-namespace", "test-running-pod", "reverseproxy")
			dateRange := meta_v1.Now()
			optionalFlag := "false"
			pmx.GetRunningPods(namespaceDirectoryName, pod, &dateRange, optionalFlag)
			file := "powermax-pod-logs/test-running-pod/test-running-pod.txt"
			data, _ := ioutil.ReadFile(file)
			got := string(data)
			if diff := cmp.Diff(got, test.expected); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expected, diff)
				return
			}
		})
	}
}

func TestGetNonRunningPods(t *testing.T) {
	type tests = []struct {
		description string
		namespace   string
		expected    string
	}

	// Unity, PowerScale, PowerStore
	var st StorageNameSpaceStruct
	var stTests = tests{
		{"get non running pod logs", "correct-namespace", "Pod status: not running"},
	}
	for _, test := range stTests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			namespaceDirectoryName := "common-pod-logs"
			pod := CreatePod(clientset, "correct-namespace", "test-nonrunning-pod", "test-container")
			st.GetNonRunningPods(namespaceDirectoryName, pod)
			file := "common-pod-logs/test-nonrunning-pod/test-container/test-nonrunning-pod.txt"

			data, _ := ioutil.ReadFile(file)
			got := string(data)
			if diff := cmp.Diff(got, test.expected); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expected, diff)
				return
			}
		})
	}

	// PowerFlex
	var pflx PowerFlexStruct
	var pflxTests = tests{
		{"get non running pod logs", "vxflexos-namespace", "Pod status: "},
	}

	for _, test := range pflxTests {
		t.Run(test.description, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()
			namespaceDirectoryName := "vxflexos-pod-logs"
			pod := CreatePod(clientset, "vxflexos-namespace", "test-nonrunning-pod", "sdc-monitor")
			pflx.GetNonRunningPods(namespaceDirectoryName, pod)
			file := "vxflexos-pod-logs/test-nonrunning-pod/sdc-monitor/test-nonrunning-pod.txt"

			data, _ := ioutil.ReadFile(file)
			got := string(data)
			if diff := cmp.Diff(got, test.expected); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expected, diff)
				return
			}
		})
	}

	// PowerMax
	var pmx PowerMaxStruct
	var pmxTests = tests{
		{"get non running pod logs", "powermax-namespace", "Pod status: "},
	}

	for _, test := range pmxTests {
		t.Run(test.description, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()
			namespaceDirectoryName := "powermax-pod-logs"
			pod := CreatePod(clientset, "powermax-namespace", "test-nonrunning-pod", "reverseproxy")
			pmx.GetNonRunningPods(namespaceDirectoryName, pod)
			file := "powermax-pod-logs/test-nonrunning-pod/reverseproxy/test-nonrunning-pod.txt"

			data, _ := ioutil.ReadFile(file)
			got := string(data)
			if diff := cmp.Diff(got, test.expected); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expected, diff)
				return
			}
		})
	}
}

func TestGetLeaseDetails(t *testing.T) {
	type tests = []struct {
		description string
		leaseName   string
		namespace   string
		podName     string
		expected    string
	}

	// PowerStore
	var ps PowerStoreStruct
	var pflxTests = tests{
		{"get lease details", "external-attacher-leader-csi-powerstore-dellemc-com", "csi-powerstore", "test-pod", "test-pod"},
	}

	for _, test := range pflxTests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			ps.namespaceName = test.namespace
			_ = CreatePod(clientset, test.namespace, test.podName, "attacher")
			_ = CreateLease(clientset, test.leaseName, test.namespace, test.podName)
			leaseHolder := ps.GetLeaseDetails()
			if diff := cmp.Diff(leaseHolder, test.expected); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expected, diff)
				return
			}
		})
	}
}

func TestGetNamespaces(t *testing.T) {
	type tests = []struct {
		description        string
		expectedNamespaces []string
	}

	var stTests = tests{
		{"get namespaces", []string{"ns-1", "ns-2"}},
	}

	for _, test := range stTests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			_ = CreateNamespace(clientset, "ns-1")
			_ = CreateNamespace(clientset, "ns-2")
			gotNamespaces := GetNamespaces()
			if diff := cmp.Diff(gotNamespaces, test.expectedNamespaces); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedNamespaces, diff)
				return
			}
		})
	}
}

func TestGetNodes(t *testing.T) {
	type tests = []struct {
		description   string
		expectedNodes []string
	}

	var stTests = tests{
		{"get nodes", []string{"10.xx.xxx.xxx", "11.xx.xxx.xxx"}},
	}

	for _, test := range stTests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			_ = CreateNodes(clientset, "10.xx.xxx.xxx")
			_ = CreateNodes(clientset, "11.xx.xxx.xxx")
			gotNodes := GetNodes()
			if diff := cmp.Diff(gotNodes, test.expectedNodes); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedNodes, diff)
				return
			}
		})
	}
}

func TestPowermaxLogs(t *testing.T) {
	type tests = []struct {
		description string
		namespace   string
		podName     string
		leaseName   string
		expected    string
	}
	currentPath, _ := os.Getwd()
	var tarCreated bool
	var pmax PowerMaxStruct
	var pmaxTests = tests{
		{"get logs of powermax", "csi-powermax", "pod1", "lease1", "tar file should be created"},
	}
	for _, test := range pmaxTests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			_ = CreateNodes(clientset, "10.xx.xxx.xxx")
			_ = CreateNodes(clientset, "11.xx.xxx.xxx")
			_ = CreateNamespace(clientset, "csi-powermax")
			_ = CreateNamespace(clientset, "ns-2")
			_ = CreatePod(clientset, test.namespace, test.podName, "attacher")
			_ = CreateLease(clientset, test.leaseName, test.namespace, test.podName)
			daysCount := -1
			pmax.GetLogs(test.namespace, "true", daysCount, 4)
			files, _ := ioutil.ReadDir(currentPath)
			for _, file := range files {
				if strings.Contains(file.Name(), test.namespace) {
					tarCreated = true
				}
			}
			if !tarCreated {
				t.Errorf("tar creation not sucessfull.")
			}
			tarCreated = false
		})
	}
}

func TestPowerscaleLogs(t *testing.T) {
	type tests = []struct {
		description string
		namespace   string
		podName     string
		leaseName   string
		expected    string
	}
	currentPath, _ := os.Getwd()
	var tarCreated bool
	var pscale PowerScaleStruct
	var pscaleTests = tests{
		{"get logs of powerscale", "csi-powerscale", "pod1", "lease1", "tar file should be created"},
	}
	for _, test := range pscaleTests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			_ = CreateNodes(clientset, "10.xx.xxx.xxx")
			_ = CreateNodes(clientset, "11.xx.xxx.xxx")
			_ = CreateNamespace(clientset, "csi-powerscale")
			_ = CreateNamespace(clientset, "ns-2")
			_ = CreatePod(clientset, test.namespace, test.podName, "attacher")
			_ = CreateLease(clientset, test.leaseName, test.namespace, test.podName)
			daysCount := -1
			pscale.GetLogs(test.namespace, "true", daysCount, 1)
			files, _ := ioutil.ReadDir(currentPath)
			for _, file := range files {
				if strings.Contains(file.Name(), test.namespace) {
					tarCreated = true
				}
			}
			if !tarCreated {
				t.Errorf("tar creation not sucessfull.")
			}
			tarCreated = false
		})
	}

}

func TestPowerflexLogs(t *testing.T) {
	type tests = []struct {
		description string
		namespace   string
		podName     string
		leaseName   string
		expected    string
	}
	currentPath, _ := os.Getwd()
	var tarCreated bool
	var pflx PowerFlexStruct
	var pflxTests = tests{
		{"get logs of powerflex", "csi-powerflex", "pod1", "lease1", "tar file should be created"},
	}
	for _, test := range pflxTests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			_ = CreateNodes(clientset, "10.xx.xxx.xxx")
			_ = CreateNodes(clientset, "11.xx.xxx.xxx")
			_ = CreateNamespace(clientset, "csi-powerflex")
			_ = CreateNamespace(clientset, "ns-2")
			_ = CreatePod(clientset, test.namespace, test.podName, "attacher")
			_ = CreateLease(clientset, test.leaseName, test.namespace, test.podName)
			daysCount := -1
			pflx.GetLogs(test.namespace, "true", daysCount, 5)
			files, _ := ioutil.ReadDir(currentPath)
			for _, file := range files {
				if strings.Contains(file.Name(), test.namespace) {
					tarCreated = true
				}
			}
			if !tarCreated {
				t.Errorf("tar creation not sucessfull.")
			}
			tarCreated = false
		})

	}
}

func TestUnityLogs(t *testing.T) {
	type tests = []struct {
		description string
		namespace   string
		podName     string
		leaseName   string
		expected    string
	}
	currentPath, _ := os.Getwd()
	var tarCreated bool
	var unity UnityStruct
	var unityTests = tests{
		{"get logs of unity", "csi-unity", "pod1", "lease1", "tar file should be created"},
	}
	for _, test := range unityTests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			_ = CreateNodes(clientset, "10.xx.xxx.xxx")
			_ = CreateNodes(clientset, "11.xx.xxx.xxx")
			_ = CreateNamespace(clientset, "csi-unity")
			_ = CreateNamespace(clientset, "ns-2")
			_ = CreatePod(clientset, test.namespace, test.podName, "attacher")
			_ = CreateLease(clientset, test.leaseName, test.namespace, test.podName)
			daysCount := -1
			unity.GetLogs(test.namespace, "true", daysCount, 2)
			files, _ := ioutil.ReadDir(currentPath)
			for _, file := range files {
				if strings.Contains(file.Name(), test.namespace) {
					tarCreated = true
				}
			}
			if !tarCreated {
				t.Errorf("tar creation not sucessfull.")
			}
			tarCreated = false
		})
	}
}

func TestPowerstoreLogs(t *testing.T) {
	type tests = []struct {
		description string
		namespace   string
		podName     string
		leaseName   string
		expected    string
	}
	currentPath, _ := os.Getwd()
	var tarCreated bool
	var pstore PowerStoreStruct
	var pstoreTests = tests{
		{"get logs of powerstore", "csi-powerstore", "pod1", "lease1", "tar file should be created"},
	}
	for _, test := range pstoreTests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			_ = CreateNodes(clientset, "10.xx.xxx.xxx")
			_ = CreateNodes(clientset, "11.xx.xxx.xxx")
			_ = CreateNamespace(clientset, "csi-powerstore")
			_ = CreateNamespace(clientset, "ns-2")
			_ = CreatePod(clientset, test.namespace, test.podName, "attacher")
			_ = CreateLease(clientset, test.leaseName, test.namespace, test.podName)
			daysCount := -1
			pstore.GetLogs(test.namespace, "true", daysCount, 3)
			files, _ := ioutil.ReadDir(currentPath)
			for _, file := range files {
				if strings.Contains(file.Name(), test.namespace) {
					tarCreated = true
				}
			}
			if !tarCreated {
				t.Errorf("tar creation not sucessfull.")
			}
			tarCreated = false
		})
	}

}

func TestPerformSanitization(t *testing.T) {
	type tests = []struct {
		description  string
		expectedFlag bool
	}
	var performSanitizationTests = tests{
		{
			"Test for perform sanitization",
			false,
		},
	}
	var st StorageNameSpaceStruct
	for _, test := range performSanitizationTests {
		t.Run(test.description, func(t *testing.T) {
			clientset = fake.NewSimpleClientset()
			namespaceDirectoryName := "pod-logs"
			pod := CreatePod(clientset, "test-namespace", "test-pod", "test-container")
			dateRange := meta_v1.Now()
			optionalFlag := "false"
			st.GetRunningPods(namespaceDirectoryName, pod, &dateRange, optionalFlag)
			actualFlag := utils.PerformSanitization(clientset, "csi_powerstore", namespaceDirectoryName)
			if diff := cmp.Diff(actualFlag, test.expectedFlag); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedFlag, diff)
				return
			}
		})
	}
}

func TestCreateTarball(t *testing.T) {
	type tests = []struct {
		description      string
		expectedFilename string
	}
	var createTarballTests = tests{
		{
			"Test for tarball creation",
			"pod-logs.tar.gz",
		},
	}
	for _, test := range createTarballTests {
		t.Run(test.description, func(t *testing.T) {
			namespaceDirectoryName := "pod-logs"
			target := "."
			createTarball(namespaceDirectoryName, target)
			filename := "pod-logs.tar.gz"
			if diff := cmp.Diff(filename, test.expectedFilename); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedFilename, diff)
				return
			}
		})
	}
}

func TestCaptureLOG(t *testing.T) {
	type tests = []struct {
		description     string
		expectedContent string
	}
	var captureLogTests = tests{
		{
			"Test for tarball creation",
			"sample data",
		},
	}
	for _, test := range captureLogTests {
		t.Run(test.description, func(t *testing.T) {
			repoName := "pod-logs"
			filename := "sample.txt"
			content := "sample data"
			captureLOG(repoName, filename, content)
			file := "pod-logs/sample.txt"

			data, _ := ioutil.ReadFile(file)
			got := string(data)
			if !strings.Contains(got, test.expectedContent) {
				t.Errorf("%T differ (-got, +want): \n\t\t - %s\n\t\t + %s", test.expectedContent, got, test.expectedContent)
				return
			}
		})
	}
}
