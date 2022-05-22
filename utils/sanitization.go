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
	// "csm-logcollector/csm"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// Logging object
var sanityLog, _ = GetLogger()

// GetSecretFilePath reads the application configuration file for identifying the complete path to secret.yaml
func GetSecretFilePath() []string {
	var secretFilePaths []string
	_, err := os.Stat("config.yml")
	if err == nil {
		yamlFile, err := ioutil.ReadFile("config.yml")
		if err != nil {
			sanityLog.Fatalf("Reading configuration file failed with error %v ", err)
		}
		data := make(map[interface{}]interface{})
		err = yaml.Unmarshal(yamlFile, data)
		if err != nil {
			sanityLog.Fatalf("Unmarshalling configuration file failed with error %v", err)
		}

		_, driverPathKey := data["driver_path"]
		if driverPathKey {
			// To access driver_path, assert data to map[interface{}]interface{}
			driverPath, ok := data["driver_path"].(map[interface{}]interface{})
			if !ok {
				sanityLog.Fatalf("driver_path is not a map!")
			}

			for key, value := range driverPath {
				// type assertion from interface{} type to string type
				value, ok1 := value.(string)
				key, ok2 := key.(string)
				if !ok1 || !ok2 {
					fmt.Printf("Please provide valid values in config.yml for key: '%s'\n", key)
					sanityLog.Fatalf("key/value is not string!")
				}

				if len(strings.TrimSpace(value)) != 0 {
					// secret.yml file relative path is same for unity, powerscale, powerstore and powermax drivers
					if key == "csi-unity" || key == "csi-powerscale" || key == "csi-powerstore" || key == "csi-powermax" {
						secretFilePath := value + "/samples/secret/secret.yaml"
						secretFilePaths = append(secretFilePaths, secretFilePath)
					} else if key == "csi-powerflex" {
						secretFilePath := value + "/samples/config.yaml"
						secretFilePaths = append(secretFilePaths, secretFilePath)
					}
				} else {
					sanityLog.Warnf("driver_path sub-key for %s is empty. Hence it's secret file can't be obtained.", key)
				}
			}
		} else {
			sanityLog.Warn("'driver_path' key not found in config.yml.")
		}
	}
	return secretFilePaths
}

// GetSecretOpted - This method will read the config file to check if getting secrets opted.
func GetSecretOpted() bool {
	var use_secrets bool
	_, err := os.Stat("config.yml")
	if err == nil {
		yamlFile, err := ioutil.ReadFile("config.yml")
		if err != nil {
			sanityLog.Fatalf("Reading configuration file failed with error %v ", err)
		}
		data := make(map[interface{}]interface{})
		err = yaml.Unmarshal(yamlFile, data)
		if err != nil {
			sanityLog.Fatalf("Unmarshalling configuration file failed with error %v", err)
		}
		_, secretsKey := data["secrets"]
		if secretsKey {
			// To access secrets, assert data to map[interface{}]interface{}
			secretsKey, ok := data["secrets"].(map[interface{}]interface{})
			if !ok {
				sanityLog.Fatalf("secrets is not a map!")
			}
			for key, value := range secretsKey {
				// type assertion from interface{} type to string type
				value, ok1 := value.(string)
				key, ok2 := key.(string)
				if !ok1 || !ok2 {
					fmt.Printf("Please provide valid values in config.yml for key: '%s'\n", key)
					sanityLog.Fatalf("key/value is not string!")
				}
				if len(strings.TrimSpace(value)) != 0 {
					// secret.yml file relative path is same for unity, powerscale, powerstore and powermax drivers
					if key == "use_secrets" {
						use_secrets = true
					}
				}
			}
		}
	}
	return use_secrets
}

// ReadSecretFileContent reads the content of secret.yaml
func ReadSecretFileContent(secretFilePaths []string) []string {
	var sensitiveContentList []string
	for item := range secretFilePaths {
		filePath := secretFilePaths[item]
		_, err := os.Stat(filePath)
		if err == nil {
			yamlFile, err := ioutil.ReadFile(filepath.Clean(filePath))
			if err != nil {
				sanityLog.Fatalf("Reading secret file %s failed with error %v ", filePath, err)
			}

			// secret/config YAML reading
			var data map[interface{}]interface{}
			var fileData string
			if strings.Contains(filePath, "powerflex") {
				// Powerflex driver has config.yaml which has data as list[map].
				fileContent, err := ioutil.ReadFile(filepath.Clean(filePath))
				fileData = string(fileContent)
				if err != nil {
					sanityLog.Fatalf("Reading secret file %s failed with error %v", filePath, err)
				}
			} else {
				data = make(map[interface{}]interface{})
				err = yaml.Unmarshal(yamlFile, data)
				if err != nil {
					sanityLog.Fatalf("Unmarshalling secret file %s failed with error %v", filePath, err)
				}
			}

			// secret/config YAML file content reading begins from here based on the respective files of the driver.
			if strings.Contains(filePath, "unity") {
				sensitiveContentList = UnitySecretContent(data, sensitiveContentList)
			}
			if strings.Contains(filePath, "powerscale") {
				sensitiveContentList = PowerscaleSecretContent(data, sensitiveContentList)
			}
			if strings.Contains(filePath, "powerstore") {
				sensitiveContentList = PowerstoreSecretContent(data, sensitiveContentList)
			}
			if strings.Contains(filePath, "powermax") {
				sensitiveContentList = PowermaxSecretContent(data, sensitiveContentList)
			}
			if strings.Contains(filePath, "powerflex") {
				sensitiveContentList = PowerflexSecretContent(fileData, sensitiveContentList)
			}
		} else {
			sanityLog.Infof("Content parsing skipped for this file, %s", err)
		}
	}
	return sensitiveContentList
}

// UnitySecretContent method reads the secret file content of unity driver
func UnitySecretContent(data map[interface{}]interface{}, sensitiveContentList []string) []string {
	_, unityDriverKeys := data["storageArrayList"]
	if unityDriverKeys {
		storageArrayList, ok := data["storageArrayList"].([]interface{})
		if !ok {
			sanityLog.Fatalf("storageArrayList is not a slice!")
		}
		sensitiveContentList = TypeConversion(storageArrayList, sensitiveContentList)
	}
	return sensitiveContentList
}

// PowerscaleSecretContent method reads the secret file content of powerscale driver
func PowerscaleSecretContent(data map[interface{}]interface{}, sensitiveContentList []string) []string {
	_, powerscaleDriverKeys := data["isilonClusters"]
	if powerscaleDriverKeys {
		isilonClusters, ok := data["isilonClusters"].([]interface{})
		if !ok {
			sanityLog.Fatalf("isilonClusters is not a slice!")
		}
		sensitiveContentList = TypeConversion(isilonClusters, sensitiveContentList)
	}
	return sensitiveContentList
}

// PowerstoreSecretContent method reads the secret file content of powerstore driver
func PowerstoreSecretContent(data map[interface{}]interface{}, sensitiveContentList []string) []string {
	_, powerstoreDriverKeys := data["arrays"]
	if powerstoreDriverKeys {
		arrays, ok := data["arrays"].([]interface{})
		if !ok {
			sanityLog.Fatalf("arrays is not a slice!")
		}
		sensitiveContentList = TypeConversion(arrays, sensitiveContentList)
	}
	return sensitiveContentList
}

// PowermaxSecretContent method reads the secret file content of powermax driver
func PowermaxSecretContent(data map[interface{}]interface{}, sensitiveContentList []string) []string {
	_, powermaxDriverKeys := data["data"]
	if powermaxDriverKeys {
		arrayData, ok1 := data["data"].(map[interface{}]interface{})
		if !ok1 {
			sanityLog.Fatalf("arrayData is not a map!")
		}
		sensitiveContentList = IdentifySensitiveContent(arrayData, sensitiveContentList)
	}
	return sensitiveContentList
}

// PowerflexSecretContent method reads the secret file content of powerflex driver and identifies the sensitive content
func PowerflexSecretContent(fileData string, sensitiveContentList []string) []string {
	fileDataList := strings.Split(fileData, "\n")
	sensitiveKeyList := []string{"arrayId", "username", "password", "endpoint", "clusterName", "globalID", "systemID", "allSystemNames", "mdm"}
	for _, str := range fileDataList {
		if containsKey(str, sensitiveKeyList) {
			tempValue1 := strings.SplitN(str, "\"", 2)
			tempValue2 := tempValue1[1]
			tempValue := strings.SplitN(tempValue2, "\"", 2)
			value := tempValue[0]
			sensitiveContentList = append(sensitiveContentList, value)
		}
	}
	return sensitiveContentList
}

func containsKey(str string, list []string) bool {
	for _, v := range list {
		if strings.Contains(str, v) {
			return true
		}
	}
	return false
}

// TypeConversion method performs the type assertion from slice to map.
// This is specifically done for Unity, PowerStore, PowerScale drivers due to slightly differnt content format of their secret.yml file.
func TypeConversion(arrayDetailsList []interface{}, sensitiveContentList []string) []string {
	for item := range arrayDetailsList {
		arrayDetailsMap, ok := arrayDetailsList[item].(map[interface{}]interface{})
		if !ok {
			sanityLog.Fatalf("arrayDetailsMap is not a map!")
		}
		sensitiveContentList = IdentifySensitiveContent(arrayDetailsMap, sensitiveContentList)
	}
	return sensitiveContentList
}

// IdentifySensitiveContent method performs the identification of sensitive content from specific drivers' secret file
func IdentifySensitiveContent(arrayDetailsMap map[interface{}]interface{}, sensitiveContentList []string) []string {
	sensitiveKeyList := []string{"arrayId", "username", "password", "endpoint", "clusterName", "globalID", "systemID", "allSystemNames", "mdm"}
	for key, value := range arrayDetailsMap {
		k, ok := key.(string)
		if !ok {
			sanityLog.Fatalf("key is not string!")
		}
		if contains(k, sensitiveKeyList) {
			v, ok := value.(string)
			if !ok {
				sanityLog.Fatalf("value is not string!")
			}
			sensitiveContentList = append(sensitiveContentList, v)
		}
	}
	return sensitiveContentList
}

func contains(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

// GetRemoteSecretFiles reads the secret/config files of remote cluster from local directory
func GetRemoteSecretFiles() []string {
	var secretFilePaths []string
	localDir := "RemoteClusterSecretFiles"
	err := filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			sanityLog.Warn(err)
			return err
		}
		if !info.IsDir() {
			fp, _ := filepath.Abs(path)
			secretFilePaths = append(secretFilePaths, fp)
		}
		return nil
	})
	if err != nil {
		sanityLog.Warn(err)
	}
	return secretFilePaths
}

// GetSecrets return all secrets in the given namespace.
func GetSecrets(clientset kubernetes.Interface, namespace string) []string {
	var secretKeys []string
	secretsList, err := clientset.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		labels.Everything()
		sanityLog.Fatalf("sanitization against sensitive contents failed with error: %s", err)
	}
	for _, secret := range secretsList.Items {
		//strData := secret.String()
		if secret.Type != "" {
			secret_name := secret.Name
			secret_user := secret.Labels["username"]
			raw_secret_password := secret.Labels["password"]
			secret_password, err := base64.StdEncoding.DecodeString(string(raw_secret_password))
			if err != nil {
				sanityLog.Fatalf("Failed to decode the secret password with error: %s", err)
			}
			fmt.Printf("\nGot Secrets for Secret Name: %s", secret_name)
			secretKeys = append(secretKeys, secret_user, string(secret_password))
		}

	}
	return secretKeys
}

// PerformSanitization method performs the sanitization of all logs files against the sensitive strings identified
func PerformSanitization(clientset kubernetes.Interface, namespace string, namespaceDirectoryName string) bool {
	var secretFilePaths []string
	var sensitiveContentList []string
	var maskingFlag = false
	if GetSecretOpted() == true {
		fmt.Print("\nGet Secrets Opted for sanitisation\n")
		sensitiveContentList = GetSecrets(clientset, namespace)
	} else {
		fmt.Print("\nGet Secrets not opted for sanitisation\n")
	}
	secretFilePaths = GetSecretFilePath()

	currentIPAddress, err := GetLocalIP()
	if err != nil {
		sanityLog.Fatalf("Error: %s", err)
	}
	remoteClusterIPAddress, clusterUsername, clusterPassword := GetRemoteClusterDetails()
	if currentIPAddress != remoteClusterIPAddress {
		if len(secretFilePaths) > 0 {
			localDirName := createDirectory("RemoteClusterSecretFiles")
			for item := range secretFilePaths {
				ScpConfigFile(secretFilePaths[item], remoteClusterIPAddress, clusterUsername, clusterPassword, localDirName)
			}
			secretFilePaths = GetRemoteSecretFiles()
		}
	}

	sanityLog.Infof("secretFilePaths: %s", secretFilePaths)
	if len(secretFilePaths) > 0 {
		sensitiveKeyList := []string{"arrayId", "username", "password", "endpoint", "clusterName", "globalID", "systemID", "allSystemNames", "mdm"}
		sanityLog.Infof("sensitiveKeyList: %s", sensitiveKeyList)
		sensitiveContentList = append(sensitiveContentList, ReadSecretFileContent(secretFilePaths)...)
		err := filepath.Walk(namespaceDirectoryName, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				sanityLog.Info(err)
				return err
			}
			if !info.IsDir() {
				for str := range sensitiveContentList {
					// read file
					fileContent, err := ioutil.ReadFile(filepath.Clean(path))
					if err != nil {
						sanityLog.Fatalf("File reading failed with error: %s", err)
					}

					var fileData = string(fileContent)
					var isValueExist bool

					// searching sensitive content with case-insensitive matching
					re := regexp.MustCompile("(?i)" + sensitiveContentList[str])
					isValueExist = re.Match(fileContent)

					// masking sensitive content with case-insensitive replacement
					if isValueExist == true {
						maskingFlag = true
						sanityLog.Infof("Sanitization initiated for file: %s", info.Name())
						fileData = re.ReplaceAllString(fileData, "*********")

						// write back to original file
						err = ioutil.WriteFile(path, []byte(fileData), 0600)
						if err != nil {
							sanityLog.Fatalf("File writing failed with error: %s", err)
							os.Exit(1)
						} else {
							sanityLog.Infof("File: %s is sanitized against the sensitive content present in drivers' secret/config YAML files", info.Name())
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			sanityLog.Infof("Error: %s", err)
		}
		if maskingFlag {
			fmt.Printf("Masking sensitive content completed.\n")
		} else {
			fmt.Printf("Sanitization not performed, either it was not opted or no sensitive content present found.\n")
		}
	}
	return maskingFlag
}
