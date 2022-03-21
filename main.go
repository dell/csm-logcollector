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
	"strconv"
	"strings"
)

var logger, _ = utils.GetLogger()
var Version = "development"

func main() {
	fmt.Printf("Version: %s\n", Version)
	logger.Info("Log started for csm-logcollector")
	fmt.Println("\n\n\tCSM Logging Tool!")
	fmt.Println("\t=================")
	fmt.Println()
	var consent string
	var namespace string
	var optionalFlag string
	var result bool
	var nsSlice []string
	var p csm.StorageNameSpace
	var err error
	var ipCount int
	const consentMsg string = "As a part of log collection, logs will be sent for further analysis. Please provide your consent.(Y/y)"

	fmt.Println(consentMsg)
	ipCount, err = fmt.Scanln(&consent)
	if (err != nil || consent != "Y" && consent != "y") || ipCount == 0 {
		fmt.Println("\nExiting the application as the user consent is not granted or invalid input")
		logger.Fatalf("Exiting the application as consent is not provided or invalid input.")
	}

	driveOption := ""
	fmt.Println("Please select the respective storage array for which CSI Driver logs need to be collected:")
	fmt.Println("1: PowerScale/Isilon\n2: Unity\n3: PowerStore\n4: PowerMax\n5: PowerFlex/VxFlexOS")
	fmt.Println("\nPlease enter your choice (e.g. enter '1' for PowerScale) :")
	ipCount, err = fmt.Scanln(&driveOption)
	driveChoice, err := strconv.Atoi(driveOption)
	if err != nil || driveChoice < 1 || driveChoice > 5 || ipCount <= 0 {
		fmt.Println("Invalid choice, please enter correct choice")
		logger.Fatalf("Entering CSI Driver choice failed")
	}

	fmt.Println("\nEnter the namespace: ")
	_, errns := fmt.Scanln(&namespace)
	if errns != nil {
		fmt.Printf("\nEntering namespace failed with error %s \n", errns.Error())
		logger.Fatalf("Entering namespace failed with error: %s", errns.Error())
	}
	temp := strings.ToLower(namespace)
	namespaces := csm.GetNamespaces()

	result, nsSlice = CheckNamespace(temp, namespaces)

	count := 4
	if !result && len(nsSlice) == 0 {
		for count > 0 {
			fmt.Println("Given namespace is not found. Please enter valid namespace:")
			_, err := fmt.Scanln(&namespace)
			if err != nil {
				logger.Fatalf("Entering valid namespace failed with error: %s", err.Error())
			}
			temp = strings.ToLower(namespace)
			result, nsSlice = CheckNamespace(temp, namespaces)
			if result || len(nsSlice) > 0 {
				break
			}
			count--
		}
	}

	CheckCount(count)

	if !result {
		count = 4
		for count > 0 {
			index := 0
			fmt.Println("Please select the correct namespace from the below choices:")
			for i, x := range nsSlice {
				fmt.Printf("%d. %s\n", i+1, x)
			}
			fmt.Println("Enter the choice:")
			_, err := fmt.Scanln(&index)
			if err != nil {
				logger.Fatalf("Entering namespace failed with error: %s", err.Error())
			}
			if index < 1 || index > len(nsSlice) {
				fmt.Println("Please select valid namespace")
				count--
			} else {
				temp = nsSlice[index-1]
				break
			}
		}
	}

	CheckCount(count)

	count = 4

	for count > 0 {
		fmt.Println("\nOptional log will be collected only when True/true is entered. Supported values are True/true/False/false.")
		ipCount, err = fmt.Scanln(&optionalFlag)
		if err != nil || ipCount <= 0 {
			fmt.Printf("Invalid input or failed to get user input. Please retry !!")
			if count <= 0 {
				logger.Fatalf("Getting Optiona log user input failed with error: %s", err.Error())
			}
		}
		if optionalFlag == "True" || optionalFlag == "true" || optionalFlag == "False" || optionalFlag == "false" {
			break
		}
		count--
	}

	CheckCount(count)

	daysUserInput := ""
	noOfDays := -1
	var intErr error

	if optionalFlag == "True" || optionalFlag == "true" {
		fmt.Println("Enter the number of days the logs need to be collected from today (to skip this filter enter 0) :")
		ipCount, inputErr := fmt.Scanln(&daysUserInput)
		noOfDays, intErr = strconv.Atoi(daysUserInput)
		if inputErr != nil || intErr != nil || noOfDays < 0 || noOfDays > 180 || ipCount <= 0 {
			fmt.Println("Invalid number of days, please enter between 1 to 180.")
			logger.Fatalf("Invalid number of days, please enter between 1 to 180.")
		}

		if noOfDays == 0 {
			noOfDays = 180
		}
		fmt.Printf("Logs will be collected for past %d days from today\n", noOfDays)
	}

	switch {
	case driveChoice == 1:
		p = csm.PowerScaleStruct{}
	case driveChoice == 2:
		p = csm.UnityStruct{}
	case driveChoice == 3:
		p = csm.PowerStoreStruct{}
	case driveChoice == 4:
		p = csm.PowerMaxStruct{}
	case driveChoice == 5:
		p = csm.PowerFlexStruct{}
	default:
		{
			fmt.Println("Invalid choice, please enter valid choice")
			logger.Fatalf("Invalid choice, exiting Application")
		}
	}

	p.GetLogs(temp, optionalFlag, noOfDays, driveChoice)
}

// CheckNamespace verifies if given namespace exists
func CheckNamespace(namespace string, namespaces []string) (bool, []string) {
	var result bool = false
	var nsSlice []string
	for _, x := range namespaces {
		if x == namespace {
			result = true
			break
		} else if strings.Contains(x, namespace) {
			nsSlice = append(nsSlice, x)
		}
	}
	return result, nsSlice
}

// CheckCount verifies if retries are exceeded
func CheckCount(count int) {
	if count == 0 {
		fmt.Printf("\nAll retries are exceeded.\n")
		logger.Fatalf("All retries are exceeded.")
	}
}
