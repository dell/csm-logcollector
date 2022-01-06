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

package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
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
					sanityLog.Infof("driver_path sub-key for %s is empty. Hence it's secret file can't be obtained.", key)
				}
			}
		} else {
			sanityLog.Info("'driver_path' key not found in config.yml.")
		}
	}
	return secretFilePaths
}

// ReadSecretFileContent reads the content of secret.yaml
func ReadSecretFileContent(secretFilePaths []string) []string {
	var sensitiveContentList []string
	for item := range secretFilePaths {
		filePath := secretFilePaths[item]
		_, err := os.Stat(filePath)
		if err == nil {
			yamlFile, err := ioutil.ReadFile(filePath)
			if err != nil {
				sanityLog.Fatalf("Reading secret file %s failed with error %v ", filePath, err)
			}

			data := make(map[interface{}]interface{})
			err = yaml.Unmarshal(yamlFile, data)
			if err != nil {
				sanityLog.Fatalf("Unmarshalling secret file failed with error %v", err)
			}

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
		}
		if os.IsNotExist(err) {
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
		sensitiveContentList = IdentifySensitiveContent(storageArrayList, sensitiveContentList)
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
		sensitiveContentList = IdentifySensitiveContent(isilonClusters, sensitiveContentList)
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
		sensitiveContentList = IdentifySensitiveContent(arrays, sensitiveContentList)
	}
	return sensitiveContentList
}

// PowermaxSecretContent method reads the secret file content of powermax driver
func PowermaxSecretContent(data map[interface{}]interface{}, sensitiveContentList []string) []string {
	_, powermaxDriverKeys := data["data"]
	if powermaxDriverKeys {
		arrayData, ok := data["data"].([]interface{})
		if !ok {
			sanityLog.Fatalf("arrays is not a slice!")
		}
		sensitiveContentList = IdentifySensitiveContent(arrayData, sensitiveContentList)
	}
	return sensitiveContentList
}

// IdentifySensitiveContent method performs the identification of sensitive content from specific driver secret file
func IdentifySensitiveContent(arrayDetailsList []interface{}, sensitiveContentList []string) []string {
	sensitiveKeyList := []string{"arrayId", "username", "password", "endpoint", "clusterName", "globalID", "systemID", "allSystemNames", "mdm"}
	for item := range arrayDetailsList {
		arrayDetailsMap, ok := arrayDetailsList[item].(map[interface{}]interface{})
		if !ok {
			sanityLog.Fatalf("arrayDetailsMap is not a map!")
		}
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

// PerformSanitization method performs the sanitization of all logs files against the sensitive strings identified
func PerformSanitization(namespaceDirectoryName string) bool {
	secretFilePaths := GetSecretFilePath()
	var maskingFlag = false
	if len(secretFilePaths) != 0 {
		sensitiveContentList := ReadSecretFileContent(secretFilePaths)
		err := filepath.Walk(namespaceDirectoryName, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				fmt.Println(err)
				return err
			}
			if !info.IsDir() {
				for str := range sensitiveContentList {
					// read file
					fileContent, err := ioutil.ReadFile(path)
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
						fileData = re.ReplaceAllString(fileData, "*********")
					}

					// write back to original file
					if err = ioutil.WriteFile(path, []byte(fileData), 0666); err != nil {
						sanityLog.Fatalf("File writing failed with error: %s", err)
						os.Exit(1)
					}
				}
			}
			return nil
		})
		if err != nil {
			fmt.Println(err)
		}
		if maskingFlag {
			fmt.Printf("Masking sensitive content completed.\n")
			sanityLog.Infof("Masking sensitive content completed.")
		} else {
			fmt.Printf("No sensitive content identified.\n")
			sanityLog.Infof("No sensitive content identified.")
		}
	}
	return maskingFlag
}
