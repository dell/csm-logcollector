package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	// "path/filepath"
	// "regexp"

	"gopkg.in/yaml.v2"
	"github.com/google/go-cmp/cmp"
	"testing"
)

func TestGetSecretFilePath(t *testing.T) {
	_, err := os.Stat("config.yml")
	if err != nil {
		fmt.Println(err)
	}
	type tests = []struct {
		description string
		expectedSecretFilePath    []string
	}
	
	var filePathTests = tests{
		{"list of secret file paths for drivers",
		[]string{
			"/root/csi-powermax/samples/secret/secret.yaml",
			"/root/csi-powerflex/samples/config.yaml",
			"/root/csi-unity/samples/secret/secret.yaml",
		 	"/root/csi-powerstore/samples/secret/secret.yaml",
			"/root/csi-powerscale/samples/secret/secret.yaml",},
		},
	}

	for _, test := range filePathTests {
		t.Run(test.description, func(t *testing.T) {
		    actual := GetSecretFilePath()
			sort.Strings(actual)
			sort.Strings(test.expectedSecretFilePath)
			if diff := cmp.Diff(actual, test.expectedSecretFilePath); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedSecretFilePath, diff)
				return
			}
		})
	}
}


func TestPowerstoreSecretContent(t *testing.T) {
	filePath := "test_data/powerstore_secret_data.yaml"
	_, err := os.Stat(filePath)
	if err != nil {
		fmt.Println(err)
	}
	yamlFile, _ := ioutil.ReadFile(filePath)
	data := make(map[interface{}]interface{})
	yaml.Unmarshal(yamlFile, data)
	type tests = []struct {
		description string
		data map[interface{}]interface{}
		sensitiveContentList []string
		expectedContentList []string
	}

	var secretContentTests = tests{
		{
			"PowerstoreSecretContent positive", data,
			[]string{},
			[]string{"unique", "sample_user","sample_password",},
		},
	}

	for _, test := range secretContentTests {
		t.Run(test.description, func(t *testing.T) {
		    // Check for powerstore secret content
			powerstoreActual := PowerstoreSecretContent(test.data, test.sensitiveContentList)
			sort.Strings(powerstoreActual)
			sort.Strings(test.expectedContentList)
			if diff := cmp.Diff(powerstoreActual, test.expectedContentList); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedContentList, diff)
				return
			}
		})
	}
}

func TestPowerscaleSecretContent(t *testing.T) {
	filePath := "test_data/powerscale_secret_data.yaml"
	_, err := os.Stat(filePath)
	if err != nil {
		fmt.Println(err)
	}
	yamlFile, _ := ioutil.ReadFile(filePath)
	data := make(map[interface{}]interface{})
	yaml.Unmarshal(yamlFile, data)
	type tests = []struct {
		description string
		data map[interface{}]interface{}
		sensitiveContentList []string
		expectedContentList []string
	}

	var secretContentTests = tests{
		{
			"PowerscaleSecretContent positive", data,
			[]string{},
			[]string{"cluster2", "user", "password", "1.2.3.4"},
		},
	}

	for _, test := range secretContentTests {
		t.Run(test.description, func(t *testing.T) {
		    // Check for powerscale secret content
			powerscaleActual := PowerscaleSecretContent(test.data, test.sensitiveContentList)
			sort.Strings(powerscaleActual)
			sort.Strings(test.expectedContentList)
			if diff := cmp.Diff(powerscaleActual, test.expectedContentList); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedContentList, diff)
				return
			}
		})
	}
}

func TestUnitySecretContent(t *testing.T) {
	filePath := "test_data/unity_secret_data.yaml"
	_, err := os.Stat(filePath)
	if err != nil {
		fmt.Println(err)
	}
	yamlFile, _ := ioutil.ReadFile(filePath)
	data := make(map[interface{}]interface{})
	yaml.Unmarshal(yamlFile, data)
	type tests = []struct {
		description string
		data map[interface{}]interface{}
		sensitiveContentList []string
		expectedContentList []string
	}

	var secretContentTests = tests{
		{
			"unitySecretContent positive", data,
			[]string{},
			// []string{},
			[]string{"ABC00000000002", "user", "password", "https://1.2.3.5/",},
		},
	}

	for _, test := range secretContentTests {
		t.Run(test.description, func(t *testing.T) {
		    // Check for unity secret content
			unityActual := UnitySecretContent(test.data, test.sensitiveContentList)
			sort.Strings(unityActual)
			sort.Strings(test.expectedContentList)
			if diff := cmp.Diff(unityActual, test.expectedContentList); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedContentList, diff)
				return
			}
		})
	}
}


func TestReadSecretFileContent(t *testing.T){
	type tests = []struct {
		description string
		secretFilePaths []string
		expectedContentList []string
	}

	var readSecretTests = tests{
		{
			"Read Secret file contents positive",
			[]string{
				"test_data/powerstore_secret_data.yaml",
				"test_data/powerscale_secret_data.yaml",
				"test_data/unity_secret_data.yaml",
				"test_data/powermax_secret_data.yaml",
			},
			[]string{"1.2.3.4", "ABC00000000002", "bm90X3RoZV91c2VybmFtZQ==", "bm90X3RoZV9wYXNzd29yZA==", "cluster2",
					 "https://1.2.3.5/", "password", "password", "sample_password", "sample_user", "unique", "user", "user",},
		},
	}
	for _, test := range readSecretTests {
		t.Run(test.description, func(t *testing.T) {
		    // Check for secret content
			fileContentResp := ReadSecretFileContent(test.secretFilePaths)
			sort.Strings(fileContentResp)
			sort.Strings(test.expectedContentList)
			if diff := cmp.Diff(fileContentResp, test.expectedContentList); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedContentList, diff)
				return
			}
		})
	}
}

func Testcontains(t *testing.T) {
	value := "powermax_secret_data"
	t.Run("String present in the list", func(t *testing.T) {
		actual_flag := contains(value ,  []string{
			"powerstore_secret_data",
			"powerscale_secret_data",
			"unity_secret_data",
			"powermax_secret_data"})
		if !actual_flag {
			t.Errorf("string not in the list of strings")
			return
		}
	})
	t.Run("String not present in the list", func(t *testing.T) {
		actual_flag := contains("invalid value" , []string{
			"powerstore_secret_data",
			"powerscale_secret_data",
			"unity_secret_data",
			"powermax_secret_data"})
		if actual_flag {
			t.Errorf("string in the list of strings")
			return
		}
	})
}

func TestPowermaxSecretContent(t *testing.T) {
	pmaxFilePath := "test_data/powermax_secret_data.yaml"
	_, err := os.Stat(pmaxFilePath)
	if err != nil {
		fmt.Println(err)
	}
	yamlFile, _ := ioutil.ReadFile(pmaxFilePath)
	data := make(map[interface{}]interface{})
	yaml.Unmarshal(yamlFile, data)
	type tests = []struct {
		description string
		data map[interface{}]interface{}
		sensitiveContentList []string
		expectedContentList []string
	}
	var secretContentTests = tests{
		{
			"PowermaxSecretContent positive", data,
			[]string{},
			[]string{"bm90X3RoZV91c2VybmFtZQ==", "bm90X3RoZV9wYXNzd29yZA==",},
		},
	}
	for _, test := range secretContentTests {
		t.Run(test.description, func(t *testing.T) {
		    // Check for powerstore secret content
			powermaxActual := PowermaxSecretContent(test.data, test.sensitiveContentList)
			sort.Strings(powermaxActual)
			sort.Strings(test.expectedContentList)
			if diff := cmp.Diff(powermaxActual, test.expectedContentList); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedContentList, diff)
				return
			}
		})
	}
}

func TestTypeConversion(t *testing.T) {
	filePath := "test_data/unity_secret_data.yaml"
	_, err := os.Stat(filePath)
	if err != nil {
		fmt.Println(err)
	}
	yamlFile, _ := ioutil.ReadFile(filePath)
	data := make(map[interface{}]interface{})
	yaml.Unmarshal(yamlFile, data)
	storageArrayList, _ := data["storageArrayList"].([]interface{})
	type tests = []struct {
		description string
		arrayDetailsList []interface{}
		sensitiveContentList []string
		expectedContentList []string
	}

	var typeConversionTests = tests{
		{
			"unitySecretContent positive", storageArrayList,
			[]string{},
			[]string{"ABC00000000002", "https://1.2.3.5/", "password", "user",},	
		},
	}

	for _, test := range typeConversionTests {
		t.Run(test.description, func(t *testing.T) {
		    // Check for unity secret content
			convertedVal := TypeConversion(test.arrayDetailsList, test.sensitiveContentList)
			sort.Strings(convertedVal)
			sort.Strings(test.expectedContentList)
			if diff := cmp.Diff(convertedVal, test.expectedContentList); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedContentList, diff)
				return
			}
		})
	}
}

func TestIdentifySensitiveContent(t *testing.T) {
	pmaxFilePath := "test_data/powermax_secret_data.yaml"
	_, err := os.Stat(pmaxFilePath)
	if err != nil {
		fmt.Println(err)
	}
	yamlFile, _ := ioutil.ReadFile(pmaxFilePath)
	data := make(map[interface{}]interface{})
	yaml.Unmarshal(yamlFile, data)
	arrayData, _ := data["data"].(map[interface{}]interface{})
	type tests = []struct {
		description string
		arrayData map[interface{}]interface{}
		sensitiveContentList []string
		expectedContentList []string
	}
	var secretContentTests = tests{
		{
			"PowermaxSecretContent positive", arrayData,
			[]string{},
			[]string{"bm90X3RoZV91c2VybmFtZQ==", "bm90X3RoZV9wYXNzd29yZA==",},
		},
	}
	for _, test := range secretContentTests {
		t.Run(test.description, func(t *testing.T) {
		    // Check for powerstore secret content
			identifyResp := IdentifySensitiveContent(test.arrayData, test.sensitiveContentList)
			sort.Strings(identifyResp)
			sort.Strings(test.expectedContentList)
			if diff := cmp.Diff(identifyResp, test.expectedContentList); diff != "" {
				t.Errorf("%T differ (-got, +want): %s", test.expectedContentList, diff)
				return
			}
		})
	}
}
