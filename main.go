/*
 Copyright © 2021 Dell Inc. or its subsidiaries. All Rights Reserved.
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

package main

import (
	"csm-logcollector/csm"
	utils "csm-logcollector/utils"
	"fmt"
	"strings"
)

var logger = utils.GetLogger()

func main() {
	logger.Info("Log started for csm-logcollector")
	fmt.Println("\n\n\tCSM Logging Tool!")
	fmt.Println("\t=================")
	fmt.Println()
	var consent string
	fmt.Println("As a part of log collection, logs will be sent for further analysis. Please provide your consent.(Y/y)")
	fmt.Scanln(&consent)
	if consent != "Y" && consent != "y" {
		logger.Fatalf("Exiting the application as consent is not provided.")
	}
	fmt.Println("Enter the namespace: ")
	var namespace string
	var optionalFlag string
	var p csm.StorageNameSpace

	fmt.Scanln(&namespace)
	temp := strings.ToLower(namespace)

	count := 4
	for count > 0 {
		fmt.Println("By default, all logs will be collected. Please enter True/true.")
		fmt.Scanln(&optionalFlag)
		if optionalFlag == "True" || optionalFlag == "true" {
			break
		}
		count--
	}

	if count == 0 {
		panic("All retries are exceeded.")
	}

	if strings.Contains(temp, "isilon") || strings.Contains(temp, "powerscale") {
		p = csm.PowerScaleStruct{}
	} else if strings.Contains(temp, "unity") {
		p = csm.UnityStruct{}
	} else if strings.Contains(temp, "powerstore") {
		p = csm.PowerStoreStruct{}
	} else if strings.Contains(temp, "powermax") {
		p = csm.PowerMaxStruct{}
	} else if strings.Contains(temp, "vxflexos") || strings.Contains(temp, "powerflex") {
		p = csm.PowerFlexStruct{}
	}

	p.GetLogs(namespace, optionalFlag)

}
