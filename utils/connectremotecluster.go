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
	"fmt"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"net"
	"os"
	"path"
	"strings"
	"time"
)

// Logging object
var remoteClusterLog, _ = GetLogger()

var remoteClusterIPAddress string

// GetLocalIP get the IP address of the current system
func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			err := ipnet.IP.To4()
			if err != nil {
				return ipnet.IP.String()
			} else {
				remoteClusterLog.Fatal(err)
			}
		}
	}
	return ""
}

// Connect method creates a connection with the remote cluster
func Connect(user, password, host string, port int) (*sftp.Client, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		sshClient    *ssh.Client
		sftpClient   *sftp.Client
		err          error
	)
	// get auth method
	auth = make([]ssh.AuthMethod, 0)
	auth = append(auth, ssh.Password(password))

	// Define the Client Config
	clientConfig = &ssh.ClientConfig{
		User:            user,
		Auth:            auth,
		Timeout:         30 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// connect to ssh
	addr = fmt.Sprintf("%s:%d", host, port)
	sshClient, err = ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		panic("Failed to dial: " + err.Error())
	}
	remoteClusterLog.Info("Successfully connected to ssh server.")

	// open an SFTP session over an existing ssh connection.
	if sftpClient, err = sftp.NewClient(sshClient); err != nil {
		return nil, err
	}

	return sftpClient, nil
}

// ScpConfigFile performs the operation to download the config file from remote cluster to container node
func ScpConfigFile(kubeconfigPath string, clusterIPAddress string, clusterUsername string, clusterPassword string) string {
	remoteClusterIPAddress = clusterIPAddress
	var (
		err        error
		sftpClient *sftp.Client
	)

	// change to the actual SSH connection user name, password, host name or IP, SSH port
	sftpClient, err = Connect(clusterUsername, clusterPassword, clusterIPAddress, 22)
	if err != nil {
		remoteClusterLog.Fatal(err)
	}
	defer sftpClient.Close()

	/*	************************************
		COPY CONFIG FILE FROM REMOTE CLUSTER
		************************************
	*/
	// remote file path and local directory path
	var remoteFilePath = kubeconfigPath
	var localDir = "."

	// open the source file
	srcFile, err := sftpClient.Open(remoteFilePath)
	if err != nil {
		remoteClusterLog.Fatal(err)
	}
	defer srcFile.Close()

	// create the destination file
	var localFileName = path.Base(remoteFilePath)
	dstFile, err := os.Create(path.Join(localDir, localFileName))
	if err != nil {
		remoteClusterLog.Fatal(err)
	}
	defer dstFile.Close()

	// copy the local directory
	if _, err = srcFile.WriteTo(dstFile); err != nil {
		remoteClusterLog.Fatal(err)
	}

	/*	**********************************************************
		COPY DRIVERS' SECRET/CONFIG YAML FILES FROM REMOTE CLUSTER
		**********************************************************
	*/

	secretFilePaths := GetSecretFilePath()
	if len(secretFilePaths) > 0 {
		for item := range secretFilePaths {
			filePath := secretFilePaths[item]
			localDirName := createDirectory("RemoteClusterSecretFiles")
			
			// open the source file
			sourceFile, err := sftpClient.Open(filePath)
			if err == nil {
				// create the destination file
				localFilePath := UpdateFileName(filePath)
				localFileName := path.Base(localFilePath)				
				destinationFile, err := os.Create(path.Join(localDirName, localFileName))
				if err != nil {
					remoteClusterLog.Fatal(err)
				}
				defer destinationFile.Close()
			
				// copy the local directory
				if _, err = sourceFile.WriteTo(destinationFile); err != nil {
					remoteClusterLog.Fatal(err)
				}
			} else {
				remoteClusterLog.Infof("Content parsing skipped for the file %s, %s", filePath, err)
			}
			defer sourceFile.Close()
		}
	}

	remoteClusterLog.Infof("Copy of %s file from remote server finished!", dstFile.Name())
	return dstFile.Name()
}

func createDirectory(name string) (dirName string) {
	_, err := os.Stat(name)

	if os.IsNotExist(err) {
		errDir := os.MkdirAll(name, 0777)
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
