```text
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2021 Intel Corporation
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

| Version   | Release Date   | OCP Version(s) compatibility | Verified on OCP         |
| --------- | ---------------| ---------------------------- | ------------------------|
| 0.0.1     | January 2021   | 4.9                          | 4.9.7                   |

# Features for Release

***v0.0.1***

- Intel Ethernet Operator
  - The operator handles the Firmware update of Intel® Ethernet Network Adapter E810 Series.
  - The operator handles the DDP (Dynamic Device Personalization) profile update of Intel® Ethernet Network Adapter E810 Series.
  - The operator handles the traffic flow configuration of Intel® Ethernet Network Adapter E810 Series.

# Changes to Existing Features

***v0.0.1***

- There are no unsupported or discontinued features relevant to this release.

# Fixed Issues

***v0.0.1***

- n/a - this is the first release.

# Known Issues and Limitations

- The installation of the Out Of Tree [ICE driver](https://www.intel.com/content/www/us/en/download/19630/29746/) is necessary for correct functionality of the operator. The provision/installation of this driver is out of scope for this operator, the user is required to provide/install the [OOT ICE driver](https://www.intel.com/content/www/us/en/download/19630/29746/intel-network-adapter-driver-for-e810-series-devices-under-linux.html) on the desired platforms.
- The creation of trusted VFs to be used by the Flow Configuration controller of the operator and the creation of VFs to be used by the applications is out of scope for this operator. The user is required to create necessary VFs.

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

***v0.0.1*** was tested using the following:

- OpenShift: 4.9.7
- OS: Red Hat Enterprise Linux CoreOS 49.84.202111022104-0
- Kubernetes:  v1.22.2+5e38c72
- NVM Package:  v1.37.13.5

# Package Versions

Package:

- Kubernetes:  v1.22.2+5e38c72
- Golang: v1.17.3
- DCF Tool: v21.08
