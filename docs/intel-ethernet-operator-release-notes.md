```text
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2020-2023 Intel Corporation
```

<!-- omit in toc -->
# Release Notes

This document provides high-level system features, issues, and limitations information for Intel® Ethernet Operator.

- [Release history](#release-history)
- [Features for Release](#features-for-release)
- [Changes to Existing Features](#changes-to-existing-features)
- [Fixed Issues](#fixed-issues)
- [Known Issues and Limitations](#known-issues-and-limitations)
- [Release Content](#release-content)
- [Hardware and Software Compatibility](#hardware-and-software-compatibility)
- [Supported Operating Systems](#supported-operating-systems)
- [Package Versions](#package-versions)

# Release history

| Version   | Release Date   | Cluster Compatibility                          | Verified on                  |
|---------- |----------------|------------------------------------------------|------------------------------|
| 0.0.1     | January  2022  | OCP 4.9                                        | OCP 4.9.7                    |
| 0.0.2     | April    2022  | BMRA 22.01(K8S v1.22.3)<br>OCP 4.10            | OCP 4.10.3                   |
| 0.1.0     | December 2022  | OCP 4.9, 4.10, 4.11                            | OCP 4.9.51, 4.10.34, 4.11.13 |
| v22.11    | December 2022  | OCP 4.9, 4.10, 4.11<br>BMRA 22.11(K8S v1.25.3) | OCP 4.9.51, 4.10.34, 4.11.13 |
| v23.07    | July     2023  | OCP 4.9, 4.10, 4.11, 4.12, 4.13                | OCP 4.12.21, 4.13.3          |
| v23.08    | August   2023  | K8S v1.25.3                                    | K8S v1.25.3                  |

# Features for Release

***v23.08***

- Allow additional parameters to be added to the nvmupdate tool through the `fwUpdateParam` field in the EthernetClusterConfig CR
- Add support for alternative firmware search path which is detected through query performed
on node by manager pod
- Add mTLS for webhook server

***v23.07***

- Add support for DDP profile with in-tree
- Certification on OCP 4.12 and 4.13
- Allow retries after a FW/DDP update failure to be switched on or off through the `retryOnFail` field in the EthernetClusterConfig CR
- Allow additional parameters to be added to the nvmupdate tool through the `fwUpdateParam` field in the EthernetClusterConfig CR

***v22.11***

- Introduced new API `ClusterFlowConfig` for cluster level Flow configuration
- Improved stability for Node FlowConfig daemon
- Add user provided certificate validation during FW/DDP package download
- Operator is updated with Operator SDK v1.25.0
- Set Min TLS v1.3 for validation webhook for improved security

***v0.1.0 (certified on OCP)***

- FW update supported on in-tree driver
- DDP profile update and traffic flow configuration are not supported when using in-tree
- Operator is updated with Operator SDK v1.25.0

***v0.0.2***

- Operator has been ported to Vanilla Kubernetes

***v0.0.1***

- Intel Ethernet Operator
  - The operator handles the Firmware update of Intel® Ethernet Network Adapter E810 Series.
  - The operator handles the DDP (Dynamic Device Personalization) profile update of Intel® Ethernet Network Adapter E810 Series.
  - The operator handles the traffic flow configuration of Intel® Ethernet Network Adapter E810 Series.

# Changes to Existing Features

***v23.08***

- Nvmupdate tool error codes 50 & 51 are no longer treated as update failures
- Alternative firmware search path is no longer enabled on OCP and disabled on K8S by default, query by manager pod is performed on node to decide variant for cluster

***v23.07***

- DDP update is now possible on in-tree driver
- Nvmupdate tool error codes 50 & 51 are no longer treated as update failures
- Reboots after a fw update no longer take place by default

***v22.11***

- Use SHA-1 instead of MD5 checksum for FW/DDP update
- Default UFT image version has been updated v22.03 -> v22.07

***v22.11***

- Use SHA-1 instead of MD5 checksum for FW/DDP update
- Default UFT image version has been updated v22.03 -> v22.07

***v0.1.0***

- FW update is now possible on in-tree driver

***v0.0.2***

- Any update of DDP packages causes node reboot
- DCF Tool has been updated v21.08 -> v22.03
- Proxy configuration for FWDDP Daemon app has been added
- Updated documentation for CRDs (EthernetClusterConfig, EthernetNodeConfig)
- Replicas of Controller Manager are now distributed accross a cluster
- EthernetClusterConfig.DrainSkip flag has been removed, IEO detects cluster type automatically and decides if drain is needed.

***v0.0.1***

- There are no unsupported or discontinued features relevant to this release.

# Fixed Issues

***v22.11***

- Fixed checksum verification for FW and DDP update
- FlowConfig daemon pod cleanup correctly
- Fixed an incorrect flow rules deletion issue

***v0.0.2***

- fixed DCF tool image registry URL reference issue. The DCF tool registry URL will be read from `IMAGE_REGISTRY` env variable during operator image build

***v0.0.1***

- n/a - this is the first release.

# Known Issues and Limitations

- The creation of trusted VFs to be used by the Flow Configuration controller of the operator and the creation of VFs to be used by the applications is out of scope for this operator. The user is required to create necessary VFs.
- If your cluster already has SR-IOV Network Operator deployed, please deploy Intel Ethernet Operator in the same namespace to avoid any issues
- The installation of the Out Of Tree [ICE driver](https://www.intel.com/content/www/us/en/download/19630/29746/) is necessary to leverage certain features of the operator. The provisioning/installation of this driver is out of scope for this operator, the user is required to provide/install the [OOT ICE driver](https://www.intel.com/content/www/us/en/download/19630/29746/intel-network-adapter-driver-for-e810-series-devices-under-linux.html) on the desired platforms. **BMRA distribution comes with required version of ICE driver and no additional steps are required.**

***v23.08***

- Operator support updates to firmware versions 3.0 or newer
- To perform fw update to 4.2 `fwUpdateParam: -if ioctl` needs to be added to `EthernetClusterConfig` CR

  ```yaml
  apiVersion: ethernet.intel.com/v1
  kind: EthernetClusterConfig
  metadata:
    name: <name>
    namespace: <namespace>
  spec:
    nodeSelectors:
      kubernetes.io/hostname: <hostname>
    deviceSelector:
      pciAddress: "<pci-address>"
    deviceConfig:
      fwURL: "<URL_to_firmware>"
      fwChecksum: "<file_checksum_SHA-1_hash>"
      fwUpdateParam: "-if ioctl"

  ```

***v23.07***

- Traffic flow configuration is not supported on in-tree driver

***v.0.1***

- The certified version 0.1.0 only functionality is fw update, DDP and traffic flow configuration is not possible on in-tree driver

# Release Content

- Intel Ethernet Operator
- Documentation

# Hardware and Software Compatibility

- [Intel® Ethernet Network Adapter E810-CQDA1/CQDA2](https://cdrdv2.intel.com/v1/dl/getContent/641676?explicitVersion=true)
- [Intel® Ethernet Network Adapter E810-XXVDA4](https://cdrdv2.intel.com/v1/dl/getContent/641676?explicitVersion=true)
- [Intel® Ethernet Network Adapter E810-XXVDA2](https://cdrdv2.intel.com/v1/dl/getContent/641674?explicitVersion=true)
- NVM utility
- OpenShift Container Platform

# Supported Operating Systems

***v23.08*** was tested using the following

- K8S v1.25.3 (Ubuntu 22.04 - 1.9.11 OOT ICE driver)

***v23.07*** was tested using the following

- Openshift
  - 4.12.21
  - 4.13.3

***v22.11***

- Intel BMRA v22.11 (Ubuntu 20.04 & 22.04)

***v0.1.0*** was tested using the following:

- OpenShift
  - 4.9.51
  - 4.10.34
  - 4.11.13

***v0.0.2*** was tested using the following:

- BMRA 22.01
- Kubernetes v1.22.3
- OS: Ubuntu 20.04.3 LTS (Focal Fossa)
- NVM Package:  v1.37.13.5

- OpenShift: 4.10.3
- OS: Red Hat Enterprise Linux CoreOS 410.84.202202251620-0 (Ootpa)
- Kubernetes:  v1.23.3+e419edf
- NVM Package:  v1.37.13.5

***v0.0.1*** was tested using the following:

- OpenShift: 4.9.7
- OS: Red Hat Enterprise Linux CoreOS 49.84.202111022104-0
- Kubernetes:  v1.22.2+5e38c72
- NVM Package:  v1.37.13.5

# Package Versions

***v23.08***

- Kubernetes: v1.25.3  
- Golang: v1.20
- DCF Tool: v22.07

***v23.07***

- Kubernetes: v1.22.2+5e38c72
- Golang: v1.19
- DCF Tool: v22.07

***v22.11***

- Kubernetes: v1.22.2+5e38c72
- Golang: v1.17.3
- DCF Tool: v22.07

***v0.0.2 Packages***

- Kubernetes: v1.22.2+5e38c72
- Golang: v1.17.3
- DCF Tool: v21.11

***v0.0.1 Packages***

- Kubernetes:  v1.22.2+5e38c72|v1.22.3
- Golang: v1.17.3
- DCF Tool: v21.08
