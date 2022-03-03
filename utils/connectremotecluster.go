/*
 Copyright (c) 2022 Dell Inc, or its subsidiaries.
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

package utils

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"gopkg.in/yaml.v2"
)

// Logging object
var remoteClusterLog, _ = GetLogger()

// GetLocalIP get the IP address of the current system
func GetLocalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", err
}

// Connect method creates a connection with the remote cluster
func Connect(user, password, host string, port int) (*sftp.Client, error) {
	var (
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		sftpClient   *sftp.Client
		err          error
	)

	// Define the Client Config
	clientConfig = &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		Timeout:         30 * time.Second,
		HostKeyCallback: trustedHostKeyCallback(""),
	}

	// connect to ssh
	addr = fmt.Sprintf("%s:%d", host, port)
	sshClient, err = ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		fmt.Println("Failed to connect with remote cluster, please verify remote cluster details and credentials")
		remoteClusterLog.Fatalf("Failed to connect with remote cluster with error %s" + err.Error())
	}
	remoteClusterLog.Info("Successfully connected to ssh server.")

	// open an SFTP session over an existing ssh connection.
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		return nil, err
	}

	return sftpClient, nil
}

// create human-readable SSH-key strings
func keyString(k ssh.PublicKey) string {
	return k.Type() + " " + base64.StdEncoding.EncodeToString(k.Marshal()) // e.g. "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY...."
}

func trustedHostKeyCallback(trustedKey string) ssh.HostKeyCallback {

	if trustedKey == "" {
		return func(_ string, _ net.Addr, k ssh.PublicKey) error {
			remoteClusterLog.Printf("WARNING: SSH-key verification is *NOT* in effect: to fix, add this trustedKey: %q", keyString(k))
			return nil
		}
	}

	return func(_ string, _ net.Addr, k ssh.PublicKey) error {
		ks := keyString(k)
		if trustedKey != ks {
			return fmt.Errorf("SSH-key verification: expected %q but got %q", trustedKey, ks)
		}

		return nil
	}
}

// ScpConfigFile performs the operation to download the config file from remote cluster to container node
func ScpConfigFile(kubeconfigPath string, clusterIPAddress string, clusterUsername string, clusterPassword string, destinationPath string) string {
	var dstinationFileName = ""
	isCopied := false
	var (
		err        error
		sftpClient *sftp.Client
	)

	// change to the actual SSH connection user name, password, host name or IP, SSH port
	sftpClient, err = Connect(clusterUsername, clusterPassword, clusterIPAddress, 22)
	if err != nil {
		remoteClusterLog.Fatalf("Error: %s", err)
	}
	defer sftpClient.Close()

	// remote file path and local directory path
	var remoteFilePath = kubeconfigPath
	var localDir = destinationPath

	// open the source file
	srcFile, err := sftpClient.Open(remoteFilePath)
	if err != nil {
		if strings.Contains(remoteFilePath, "samples") || strings.Contains(remoteFilePath, "secret") {
			remoteClusterLog.Infof("Content parsing skipped for the file %s, %s", remoteFilePath, err)
		} else {
			fmt.Printf("Failed to read file: %s with error %s \n", remoteFilePath, err.Error())
			remoteClusterLog.Fatalf("Failed to read file: %s with error %s", remoteFilePath, err.Error())
		}
	} else {
		// create the destination file
		if strings.Contains(remoteFilePath, "samples") || strings.Contains(remoteFilePath, "secret") {
			remoteFilePath = UpdateFileName(remoteFilePath)
		}
		localFileName := path.Base(remoteFilePath)
		createFilePath := path.Join(localDir, localFileName)
		dstFile, err := os.Create(filepath.Clean(createFilePath))
		if err != nil {
			remoteClusterLog.Fatalf("Error: %s", err)
		}
		dstinationFileName = dstFile.Name()

		defer func() {
			if err := dstFile.Close(); err != nil {
				remoteClusterLog.Fatalf("Error closing file: %s with error %s \n", dstinationFileName, err.Error())
			}
		}()

		// copy the local directory
		if _, err = srcFile.WriteTo(dstFile); err != nil {
			remoteClusterLog.Fatalf("Error: %s", err)
		} else {
			isCopied = true
		}
	}
	if srcFile != nil {
		defer srcFile.Close()
	}
	if isCopied {
		remoteClusterLog.Infof("Copy of %s file from remote server finished!", kubeconfigPath)
	}
	return dstinationFileName

}

func createDirectory(name string) (dirName string) {
	_, err := os.Stat(name)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(name, 0750)
		if errDir != nil {
			remoteClusterLog.Fatalf("Error while creating directory: %s", err.Error())
		}
	}
	return name
}

// UpdateFileName method suffixes the driver name along with secret/config file of driver
func UpdateFileName(filePath string) string {
	var secretFilePath string
	if strings.Contains(filePath, "unity") {
		str := strings.SplitN(filePath, ".", 2)
		secretFilePath = str[0] + "-unity." + str[1]
	}
	if strings.Contains(filePath, "powerscale") {
		str := strings.SplitN(filePath, ".", 2)
		secretFilePath = str[0] + "-powerscale." + str[1]
	}
	if strings.Contains(filePath, "powerstore") {
		str := strings.SplitN(filePath, ".", 2)
		secretFilePath = str[0] + "-powerstore." + str[1]
	}
	if strings.Contains(filePath, "powermax") {
		str := strings.SplitN(filePath, ".", 2)
		secretFilePath = str[0] + "-powermax." + str[1]
	}
	if strings.Contains(filePath, "powerflex") {
		str := strings.SplitN(filePath, ".", 2)
		secretFilePath = str[0] + "-powerflex." + str[1]
	}
	return secretFilePath
}

// GetRemoteClusterDetails get the IP address of the remote cluster
func GetRemoteClusterDetails() (string, string, string) {
	var userName string
	var password string
	var ipAddrr string
	_, err := os.Stat("config.yml")
	if err == nil {
		yamlFile, err := ioutil.ReadFile("config.yml")
		if err != nil {
			remoteClusterLog.Fatalf("Reading configuration file failed with error %v ", err)
		}

		data := make(map[interface{}]interface{})
		err = yaml.Unmarshal(yamlFile, data)
		if err != nil {
			remoteClusterLog.Fatalf("Unmarshalling configuration file failed with error %v", err)
		}

		for k := range data {
			if k == "kubeconfig_details" {
				// To access kubeconfig_details, assert type of data["kubeconfig_details"] to map[interface{}]interface{}
				kubeconfigDetails, ok := data["kubeconfig_details"].(map[interface{}]interface{})
				if !ok {
					remoteClusterLog.Fatalf("kubeconfig_details is not a map!")
				}

				for key, value := range kubeconfigDetails {
					// type assertion from interface{} type to string type
					key, ok1 := key.(string)
					value, ok2 := value.(string)
					if !ok1 || !ok2 {
						remoteClusterLog.Fatalf("key/value is not string!")
					}
					if len(strings.TrimSpace(value)) != 0 {
						if key == "ip_address" {
							ipAddrr = value
						}
						if key == "username" {
							userName = value
						}
						if key == "password" {
							password = value
						}
					} else {
						remoteClusterLog.Fatalf("No value found for kubeconfig_details sub-key: %s", key)
					}
				}
			}
		}
	}
	return ipAddrr, userName, password
}
