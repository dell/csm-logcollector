<!--
Copyright (c) 2022 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Dell Container Storage Modules (CSM) Log Collector

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0%20adopted-ff69b4.svg)](docs/CODE_OF_CONDUCT.md)
[![License](https://img.shields.io/github/license/dell/csm)](LICENSE)


Dell Container Storage Modules (CSM) Log Collector is an open-source application designed to collect the logs of Dell CSM and CSI drivers.
Currently, the logs of CSI drivers can only be collected.


## Supported Platforms
   | **Log collector** | **CSI Drivers** | **Operating System**|
|---------------------|-----------------------|------------------------------|
| v1.1.0 | PowerMax <br> PowerScale <br> PowerStore <br> Unity <br> VxflexOs| SLES 15.3  <br> RHEL 8.4 |

## Support
For any CSM log collector issues, questions or feedback, please follow our [support process](https://github.com/dell/csm/blob/main/docs/SUPPORT.md)

## Installing Application
 
  1. To install through docker image, please follow the below mentioned steps.

    docker pull quay.io/arindam_datta/csm-logcollector:latest

  2. Browse through the docker image and navigate to the folder '/root/csm-logcollector'. Folder contains one binary file(csm-logcollector) and one configuration file(config.yml).
  config.yml file must be updated with the required details as explained in the [configuration](#Configuration) section.

  3. Alternatively, Clone the repo using the command:

    git clone https://github.com/dell/csm-logcollector/tree/1.1.0

  4. Go to the root directory of go.mod

  5. Execute the following command to install run-time dependencies:

    go mod tidy

  6. To execute the application, please refer [Using Application](#using-application) section.

## Configuration
  1. The config.yml contains generic configuration which are necessary to execute the application.
  
  2. The config.yml should be located at the root folder of the application.
  Each item in this file is described below. 

 * <b>kubeconfig_details</b>: Includes the Kubernetes configuration file path, Cluster IP and credentials required to connect to the Kubernetes cluster. It is a mandatory parameter which specifies the details about remote Kubernetes cluster. It includes following sub-fields.
      * path: The absolute path of the Kubernetes config file. If not specified, by default, application will look for config file at <home_directory_of_user>/.kube folder.
      * ip_address: The IP address of the remote Kubernetes cluster.
      * username: The username required to connect to the remote Kubernetes cluster.
      * password: The password required to connect to the remote Kubernetes cluster.

  3. <b>destination_path</b>: Destination path where tarball is to be copied. It is an optional parameter.

  4. <b>driver_path</b>: Path where CSI driver is installed in the Kubernetes cluster for the respective storage platform. This is optional field. If not provided, log sanitization will be skipped. It includes following sub-fields.
      * csi-unity: CSI driver path for Unity.
      * csi-powerstore: CSI driver path for PowerStore.
      * csi-powerscale: CSI driver path for PowerScale.
      * csi-powermax: CSI driver path for PowerMax.
      * csi-powerflex: CSI driver path for PowerFlex.

## Using Application
  1. To run the application in the container, navigate to the root folder and run the following command:

    ./csm-logcollector

  2. If the repo is cloned from the source, run the following command:

    go run main.go


## About

Dell Container Storage Modules (CSM) Log Collection application is a completely open source and community-driven application. All components are available
under [Apache 2 License](https://www.apache.org/licenses/LICENSE-2.0.html) on
GitHub.
