package utils

import (
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGetLocalIP(t *testing.T) {
	conn, _ := net.Dial("udp", "1.2.3.4:80")
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	expectedAddr := (localAddr.IP).String()
	t.Run("Get local ip positive", func(t *testing.T) {
		actual, _ := GetLocalIP()
		if diff := cmp.Diff(actual, expectedAddr); diff != "" {
			t.Errorf("%T differ (-got, +want): %s", expectedAddr, diff)
			return
		}
	})

}

func TestUpdateFileName(t *testing.T) {
	type tests = []struct {
		description      string
		inputFilePath    string
		expectedFilePath string
	}

	var updateFilePathTests = tests{
		{
			"Update unity secret file's name", "/root/csi-unity/samples/secret/secret.yaml",
			"/root/csi-unity/samples/secret/secret-unity.yaml",
		},
		{
			"Update powermax secret file's name", "/root/csi-powermax/samples/secret/secret.yaml",
			"/root/csi-powermax/samples/secret/secret-powermax.yaml",
		},
		{
			"Update powerflex secret file's name", "/root/csi-powerflex/samples/config.yaml",
			"/root/csi-powerflex/samples/config-powerflex.yaml",
		},
		{
			"Update powerstore secret file's name", "/root/csi-powerstore/samples/secret/secret.yaml",
			"/root/csi-powerstore/samples/secret/secret-powerstore.yaml",
		},
		{
			"Update powerscale secret file's name", "/root/csi-powerscale/samples/secret/secret.yaml",
			"/root/csi-powerscale/samples/secret/secret-powerscale.yaml",
		},
	}

	for _, test := range updateFilePathTests {
		t.Run(test.description, func(t *testing.T) {
			// Check for powerstore secret content
			actualFilePath := UpdateFileName(test.inputFilePath)
			fmt.Println(actualFilePath)
			if diff := cmp.Diff(actualFilePath, test.expectedFilePath); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedFilePath, diff)
				return
			}
		})
	}
}

func TestCreateDirectory(t *testing.T) {
	t.Run("Create directory if do not exist", func(t *testing.T) {
		_ = createDirectory("sample_directory")
		if _, err := os.Stat("sample_directory"); err != nil {
			if os.IsNotExist(err) {
				t.Errorf("Directory do not exist")
			}
		} else {
			currentPath, _ := os.Getwd()
			os.RemoveAll(currentPath + "/sample_directory")
		}
	})
}
