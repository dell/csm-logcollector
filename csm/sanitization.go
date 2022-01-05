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

package csm

import (
	utils "csm-logcollector/utils"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

// Logging object
var sanityLog, _ = utils.GetLogger()

// GetSecretFilePath reads the application configuration file for identifying the complete path to secret.yaml
func GetSecretFilePath(namespace string) []string {
	var secretFilePaths []string
	_, err := os.Stat("config.yml")
	if err == nil {
		yamlFile, err := ioutil.ReadFile("config.yml")
		if err != nil {
			sanityLog.Fatalf("Reading configuration file failed with error %v ", err)
		}
		// defer yamlFile.Close()
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
				value, ok := value.(string)
				if !ok {
					sanityLog.Fatalf("value is not string!")
				}
				// type assertion from interface{} type to string type
				key, ok := key.(string)
				if !ok {
					sanityLog.Fatalf("key is not string!")
				}

				if key == "csi-unity" { // csi-unity driver
					secretFilePath := value + "/samples/secret/secret.yaml"
					secretFilePaths = append(secretFilePaths, secretFilePath)
				} else if key == "csi-powerscale" { // csi-powerscale/isilon driver
					secretFilePath := value + "/samples/secret/secret.yaml"
					secretFilePaths = append(secretFilePaths, secretFilePath)
				} else if key == "csi-powerstore" { // csi-powerstore driver
					secretFilePath := value + "/samples/secret/secret.yaml"
					secretFilePaths = append(secretFilePaths, secretFilePath)
				}
			}
		} else {
			sanityLog.Info("'driver_path' key not found in config.yml.")
		}
	}
	return secretFilePaths
}

// readSecretFileContent reads the content of secret.yaml
func readSecretFileContent(secretFilePaths []string) []string {
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

			_, unityDriverKeys := data["storageArrayList"]
			if unityDriverKeys {
				storageArrayList, ok := data["storageArrayList"].([]interface{})
				if !ok {
					sanityLog.Fatalf("storageArrayList is not a slice!")
				}
				sensitiveContentList = identifySensitiveContent(storageArrayList, sensitiveContentList)
			}

			_, powerscaleDriverKeys := data["isilonClusters"]
			if powerscaleDriverKeys {
				isilonClusters, ok := data["isilonClusters"].([]interface{})
				if !ok {
					sanityLog.Fatalf("isilonClusters is not a slice!")
				}
				sensitiveContentList = identifySensitiveContent(isilonClusters, sensitiveContentList)
			}
			_, powerstoreDriverKeys := data["arrays"]
			if powerstoreDriverKeys {
				arrays, ok := data["arrays"].([]interface{})
				if !ok {
					sanityLog.Fatalf("arrays is not a slice!")
				}
				sensitiveContentList = identifySensitiveContent(arrays, sensitiveContentList)
			}
			_, powermaxDriverKeys := data["data"]
			if powermaxDriverKeys {
				arrayData, ok := data["data"].([]interface{})
				if !ok {
					sanityLog.Fatalf("arrays is not a slice!")
				}
				sensitiveContentList = identifySensitiveContent(arrayData, sensitiveContentList)
			}
		}
	}
	sanityLog.Infof("Sensitive content identified: %s", sensitiveContentList)
	return sensitiveContentList
}

func identifySensitiveContent(arrayList []interface{}, sensitiveContentList []string) []string {
	sensitiveKeyList := []string{"arrayId", "username", "password", "endpoint", "clusterName", "globalID", "systemID", "allSystemNames", "mdm"}
	for item := range arrayList {
		detailsMap, ok := arrayList[item].(map[interface{}]interface{})
		if !ok {
			sanityLog.Fatalf("detailsMap is not a map!")
		}
		for key, value := range detailsMap {
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

func performSanitization(sensitiveContentList []string, namespaceDirectoryName string) {
	var maskingFlag = false
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

				// lowercase conversion
				var fileData = strings.ToLower(string(fileContent))
				var isValueExist bool

				isValueExist, err = regexp.Match(sensitiveContentList[str], fileContent)
				if err != nil {
					sanityLog.Fatalf("Regex matching for value: %s failed with error: %s", sensitiveContentList[str], err)
				}

				// file identification if key:value both exists in file
				if isValueExist == true {
					maskingFlag = true
					sanityLog.Infof("File: %s contains %s", info.Name(), sensitiveContentList[str])
					// masking sentsitve content
					fileData = strings.Replace(fileData, sensitiveContentList[str], "*********", -1)
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
	} else {
		fmt.Printf("No senstive content identifed.\n")
	}
}
