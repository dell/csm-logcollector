<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Dell EMC Container Storage Modules (CSM) Log Collector

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0%20adopted-ff69b4.svg)](docs/CODE_OF_CONDUCT.md)
[![License](https://img.shields.io/github/license/dell/csm)](LICENSE)


Dell EMC Container Storage Modules (CSM) Log Collector is an open-source application designed to collect the logs of Dell EMC CSM products.

## Supported Platforms
   | **Log collector** | **CSI Drivers** | **Operating System**|
|---------------------|-----------------------|------------------------------|
| v1.0.0 | PowerMax <br> PowerScale <br> PowerStore <br> Unity <br> VxflexOs|Ubuntu 18.04  <br> RHEL 8.4 |

## Support
For any CSM log collector issues, questions or feedback, please follow our [support process](https://github.com/dell/csm/blob/main/docs/SUPPORT.md)

## Installing Application
  1. Clone the repo using the command:

    git clone https://github.com/dell/csm-logcollector/tree/1.0.0

  2. Go to the root directory of go.mod .
  3. Execute the following command to install dependencies:

    go mod tidy

## Configuration
  1. The config.yml contains configuration details related to Kubernetes cluster, path where tarball is to be copied and CSI driver path. The config.yml should be located at the root folder of the application.

  2. <b>kubeconfig_details</b>: Includes the Kubernetes configuration file path, Cluster IP and credentials required to connect to the Kubernetes cluster. Mandatory parameter while connecting to remote Kubernetes cluster. It can include following sub-fields.
      * path: The path of the kubeconfig file. If not specified, by default, application will look for config file at <home_directory_of_user>/.kube folder.
      * ip_address: The IP address of the remote Kubernetes cluster.
      * username: The username required to connect to the remote Kubernetes cluster.
      * password: The password required to connect to the remote Kubernetes cluster.

  3. <b>destination_path</b>: Destination path where tarball is to be copied. It is an optional parameter.

  4. <b>driver_path</b>: Path where CSI driver repo is installed in the users system for the respective storage platform. This path will help to identify the relative path to the particular drivers secret file, which is utilized for log content sanitization. This is optional field. If not provided, log sanitization will be skipped. It can include following sub-fields.
      * csi-unity: CSI repo path for Unity.
      * csi-powerstore: CSI repo path for PowerStore.
      * csi-powerscale: CSI repo path for PowerScale.
      * csi-powermax: CSI repo path for PowerMax.
      * csi-powerflex: CSI repo path for PowerFlex.

## Using Application
  1. Once the dependencies are installed, run the following command:

    go run main.go

## Runtime Dependencies
<b>client-go</b> library is needed to use the application.

## About

Dell EMC Container Storage Modules (CSM) Log Collection application is 100% open source and community-driven. All components are available
under [Apache 2 License](https://www.apache.org/licenses/LICENSE-2.0.html) on
GitHub.
