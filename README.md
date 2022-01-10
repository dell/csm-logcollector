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
  1. The config.yml contains configuration details related to Kubernetes cluster, path where tarball is to be copied and CSI driver path.

  2. <b>kubeconfig_details</b>: Includes the Kubernetes configuration file path, Cluster IP and credentials required to connect to the Kubernetes cluster. Mandatory parameter while connecting to remote Kubernetes cluster.

  3. <b>destination_path</b>: Destination path where tarball is to be copied. It is an optional parameter.

  4. <b>driver_path</b>: Path where CSI repo is cloned for respective storage platform. This is optional field. If not provided, log sanitization will be skipped. It can include following sub-fields.
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
