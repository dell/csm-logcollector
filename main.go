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
	fmt.Println("Enter the namespace: ")
	var namespace string
	var optionalFlag string
	var p csm.StorageNameSpace

	fmt.Scanln(&namespace)
	temp := strings.ToLower(namespace)

	fmt.Println("Specify optional logs needs to be collected(true):")
	fmt.Scanln(&optionalFlag)

	if optionalFlag != "true" {
		fmt.Println("optionalFlag is set to true by default.")
	}

	optionalFlag = "true"

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
