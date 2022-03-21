<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

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
   | **Log collector** | **CSI Drivers** | **Operating System**|**Kubernetes**|
|---------------------|-----------------------|------------------------------|------------------------------|
| v1.0.0 | PowerMax, <br> PowerScale, <br> PowerStore, <br> Unity, <br> PowerFlex| RHEL 8.4, <br> SLES 15.3, <br> Ubuntu 18.04| 1.21, <br> 1.22

## Support
Please interact with us on [GitHub](https://github.com/dell/csm-logcollector) by creating a [GitHub Issue](https://github.com/dell/csm-logcollector/issues) for any CSM log collector issues, questions or feedback.

## Prerequisites
  
  * go 1.17.2
  
## Installing application via docker image
 
  1. To install through docker image, please follow the below mentioned steps.

     **Note**:As a prerequisite docker should be installed & running.

    docker pull dellemc/csm-log-collector

  2. Run the docker image.
        
    docker run -it dellemc/csm-log-collector 
    
  3. Browse through the docker image and navigate to the folder '/root/csm-logcollector'. Folder contains one binary file(csm-logcollector) and one configuration file(config.yml).
  
  4. config.yml file must be updated with the required details as explained in the [configuration](#Configuration) section.
  
  5. To execute the application please refer [Using Application](#using-application) section.
  
  6. Upon successful execution archive containing the logs will be created in the desired folder with latest date time stamp. Now this archieve can be copied to required system from here.
  
  7. After the archieve is copied, this docker container can be exited using below command.
  
    exit

## Installing application via GitHub

  1. Alternatively, Clone the repo using the command:

    git clone https://github.com/dell/csm-logcollector/tree/1.0.0

  2. Go to the root directory of go.mod

  3. Execute the following command to install run-time dependencies:

    go mod tidy
  
  4. config.yml file must be updated with the required details as explained in the [configuration](#Configuration) section.
  
  5. To execute the application, please refer [Using Application](#using-application) section.

## Offline installation

  1. Download the docker image on a system that has internet access. 

    docker pull dellemc/csm-log-collector

  2. Save the Docker image as a tar file
  
    docker save -o <tar-filename>.tar dellemc/csm-log-collector

  3. Copy the image to other system with regular file transfer tools such as cp, scp, etc.
  
    scp <tar-filename>.tar root@10.xxx.xxx.xxx:<path-for-tar-file>
    
  _Note: User might need to give credentials for successful authentication._
  
  4. Load the image into Docker:

    docker load -i <path-for-tar-file>/<tar-filename>.tar

  5. Then follow the same steps from step 2 onwards as mentioned [here](#installing-application-via-docker-image) for execution.
  
## Configuration
  1. The config.yml contains generic configuration which are necessary to execute the application.
  
  2. The config.yml should be located at the root folder of the application.
  Each item in this file is described below. 

 * <b>kubeconfig_details</b>: Includes the Kubernetes configuration file path, Cluster IP and credentials required to connect to the Kubernetes cluster. It is a mandatory parameter which specifies the details about remote Kubernetes cluster. It includes following sub-fields.
      * path: The absolute path of the Kubernetes config file. If not specified, by default, application will look for config file at <home_directory_of_user>/.kube folder.
      * ip_address: The IP address of the remote Kubernetes cluster.
      * username: The username required to connect to the remote Kubernetes cluster.
      * password: The password required to connect to the remote Kubernetes cluster.

  3. <b>destination_path</b>: Destination path where tarball is to be copied. It is an optional parameter. If not given then the tarball will be generated at the root location of the tool.

  4. <b>driver_path</b>: Path where CSI driver is installed in the Kubernetes cluster for the respective storage platform. This is optional field and required only if log sanitization is to be performed. Any sensitive data like credentials, ip, fqdn etc. present in the files pointing to below mentioned paths will be masked. It includes following sub-fields.
      * csi-unity: CSI driver path for Unity.
      * csi-powerstore: CSI driver path for PowerStore.
      * csi-powerscale: CSI driver path for PowerScale.
      * csi-powermax: CSI driver path for PowerMax.
      * csi-powerflex: CSI driver path for PowerFlex.

## Using Application
  * To run the application in the container, navigate to the '/root/csm-logcollector' folder and run the following command:

        ./csm-logcollector

  * If the repo is cloned from the source, run the following command:

        go run main.go

## Features
* The log collector application collects the following logs from the cluster:
    * List of all namespaces.
    * Get pods in a namespace.
    * Describe nodes in a cluster.
    * Describe pod in a namespace.
* When the optional logs option is passed as True then the following will be added into the logs:
    * Describe pvc in a namespace.
    * Date filter to get the logs of past 180 days at max.
    * Describe running pod in namespace.
    
## About

Dell Container Storage Modules (CSM) Log Collection application is completely open source and community-driven application. All components are available
under [Apache 2 License](https://www.apache.org/licenses/LICENSE-2.0.html) on GitHub.
