```text
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2020-2022 Intel Corporation
```
<!-- omit in toc -->
# Intel Ethernet Operator documentation

- [Overview](#overview)
- [Intel Ethernet Operator](#intel-ethernet-operator)
  - [Intel Ethernet Operator - Controller-Manager](#intel-ethernet-operator---controller-manager)
    - [Alternative firmware search path on nodes with /lib/firmware read-only](#alternative-firmware-search-path-on-nodes-with-libfirmware-read-only)
  - [Intel Ethernet Operator - Device Discovery](#intel-ethernet-operator---device-discovery)
  - [Intel Ethernet Operator - FW/DDP Daemon](#intel-ethernet-operator---fwddp-daemon)
    - [Firmware Update (FW) Functionality](#firmware-update-fw-functionality)
    - [Dynamic Device Personalization (DDP) Functionality](#dynamic-device-personalization-ddp-functionality)
  - [Intel Ethernet Operator - Flow Configuration](#intel-ethernet-operator---flow-configuration)
    - [Node Flow Configuration Controller](#node-flow-configuration-controller)
    - [Unified Flow Tool](#unified-flow-tool)
  - [Prerequisites](#prerequisites)
    - [Intel Ethernet Operator - SRIOV](#intel-ethernet-operator---sriov)
    - [Node Feature Discovery](#node-feature-discovery)
- [Deploying the Operator](#deploying-the-operator)
  - [Installing Go](#installing-go)
  - [Installing Operator SDK](#installing-operator-sdk)
  - [Applying custom resources](#applying-custom-resources)
    - [Webserver for disconnected environment](#webserver-for-disconnected-environment)
    - [Updating Firmware](#updating-firmware)
    - [Updating DDP](#updating-ddp)
    - [Deploying Flow Configuration Agent](#deploying-flow-configuration-agent)
      - [Creating Trusted VF using SRIOV Network Operator](#creating-trusted-vf-using-sriov-network-operator)
      - [Check node status](#check-node-status)
      - [Create DCF capable SRIOV Network](#create-dcf-capable-sriov-network)
      - [Build UFT image](#build-uft-image)
      - [Creating FlowConfig Node Agent Deployment CR](#creating-flowconfig-node-agent-deployment-cr)
      - [Verifying that FlowConfig Daemon is running on available nodes:](#verifying-that-flowconfig-daemon-is-running-on-available-nodes)
      - [Creating Flow Configuration rules with Intel Ethernet Operator](#creating-flow-configuration-rules-with-intel-ethernet-operator)
      - [Update a sample Node Flow Configuration rule](#update-a-sample-node-flow-configuration-rule)
- [Hardware Validation Environment](#hardware-validation-environment)
- [Summary](#summary)

## Overview

This document provides the instructions for using the Intel Ethernet Operator on supported Kubernetes clusters (Vanilla K8s or Red Hat's OpenShift Container Platform). This operator was developed with aid of the Operator SDK project.

## Intel Ethernet Operator

The role of the Intel Ethernet Operator is to orchestrate and manage the configuration of the capabilities exposed by the Intel E810 Series network interface cards (NICs). The operator is a state machine which will configure certain functions of the card and then monitor the status and act autonomously based on the user interaction.
The operator design of the Intel Ethernet Operator supports the following E810 series cards:

- [Intel® Ethernet Network Adapter E810-CQDA1/CQDA2](https://cdrdv2.intel.com/v1/dl/getContent/641671?explicitVersion=true)
- [Intel® Ethernet Network Adapter E810-XXVDA4](https://cdrdv2.intel.com/v1/dl/getContent/641676?explicitVersion=true)
- [Intel® Ethernet Network Adapter E810-XXVDA2](https://cdrdv2.intel.com/v1/dl/getContent/641674?explicitVersion=true)

The Intel Ethernet Operator provides functionality for:

- Update of the devices' FW (Firmware) via [NVM Update tool](https://www.intel.com.au/content/www/au/en/support/articles/000088453/ethernet-products.html).
- Update of the devices' DDP ([Dynamic Device Personalization](https://www.intel.com/content/www/us/en/architecture-and-technology/ethernet/dynamic-device-personalization-brief.html)) profile.
- Flow configuration of traffic handling for the devices, based on supported DDP profile.

Upon deployment the operator provides APIs, Controllers and Daemons responsible for management and execution of the supported features. A number of dependencies (ICE driver, SRIOV Network Operator, NFD) must be fulfilled before the deployment of this operator - these dependencies are listed in the [prerequisites section](#prerequisites). The user interacts with the operator by providing CRs (CustomResources). The operator constantly monitors the state of the CRs to detect any changes and acts based on the changes detected. There is a separate CR to be provided for the FW/DDP update functionality and the Flow Configuration functionality. Once the CR is applied or updated, the operator/daemon checks if the configuration is already applied and if it is not, it applies the configuration.

![Intel Ethernet Operator Design](images/Diagram1.png)

### Intel Ethernet Operator - Controller-Manager

The controller manager pod is the first pod of the operator, it is responsible for deployment of other assets, exposing the APIs, handling of the CRs and executing the validation webhook. It contains the logic for accepting and splitting the FW/DDP CRs into node CRs and reconciling the status of each CR.

The validation webhook of the controller manager is responsible for checking each CR for invalid arguments.

#### Alternative firmware search path on nodes with /lib/firmware read-only

Some orchestration platforms based on kubernetes have /lib/firmware directory immutable. Intel Ethernet Operator needs read and write permissions in this directory to perform FW and DDP updates. Solution to this problem is [alternative firmware search path](https://docs.kernel.org/driver-api/firmware/fw_search_path.html#:~:text=There%20is%20an%20alternative%20to,module%2Ffirmware_class%2Fparameters%2Fpath). Custom firmware path needs be set up on all nodes in the cluster. Controller manager pod checks content of `/sys/module/firmware_class/parameters/path` on node on which it was deployed and takes that path into consideration while managing rest of the operator resources.

### Intel Ethernet Operator - Device Discovery

The CLV-discovery pod is a DaemonSet deployed on each worker node in the cluster. It's responsibility is to check if a supported hardware is discovered on the platform and label the node accordingly.

To get all the nodes containing the supported devices run:

```shell
kubectl get EthernetNodeConfig -A

NAMESPACE                 NAME       UPDATE
intel-ethernet-operator   worker-1   InProgress
intel-ethernet-operator   worker-2   InProgress
```

To get the list of supported devices to be found by the discovery pod run:

```shell
kubectl describe configmap supported-clv-devices -n intel-ethernet-operator
```

### Intel Ethernet Operator - FW/DDP Daemon

The FW/DDP daemon pod is a DaemonSet deployed as part of the operator. It is deployed on each node labeled with appropriate label indicating that a supported E810 Series NIC is detected on the platform. It is a reconcile loop which monitors the changes in each node's `EthernetNodeConfig` and acts on the changes. The logic implemented into this Daemon takes care of updating the cards' NIC firmware and DDP profile. It is also responsible for draining the nodes, taking them out of commission and rebooting when required by the update.

#### Firmware Update (FW) Functionality

Once the operator/daemon detects a change to a CR related to the update of the Intel® E810 NIC firmware, it tries to perform an update. The firmware for the Intel® E810 NICs is expected to be provided by the user in form of a `tar.gz` file. The user is also responsible to verify that the firmware version is compatible with the device. The user is required to place the firmware on an accessible HTTP server and provide an URL for it in the CR. If the file is provided correctly and the firmware is to be updated, the Ethernet Configuration Daemon will update the Intel® E810 NICs with the NVM utility provided.

To update the NVM firmware of the Intel® E810 cards' NICs user must create a CR containing the information about which card should be programmed. The Physical Functions of the NICs will be updated in logical pairs. The user needs to provide the FW URL and checksum (SHA-1) in the CR.

For a sample CR go to [Updating Firmware](#updating-firmware).

#### Dynamic Device Personalization (DDP) Functionality

Once the operator/daemon detects a change to a CR related to the update of the Intel® E810 DDP profile, it tries to perform an update. The DDP profile for the Intel® E810 NICs is expected to be provided by the user. The user is also responsible to verify that the DDP version is compatible with the device. The user is required to place the DDP package on an accessible HTTP server and provide an URL for it in the CR. If the file is provided correctly and the DDP is to be updated, the Ethernet Configuration Daemon will update the DDP profile of Intel® E810 NICs by placing it in correct filesystem on the host.

To update the DDP profile of the Intel® E810 NIC user must create a CR containing the information about which card should be programmed. All the Physical Functions of the NICs will be updated for each NIC.

For a sample CR go to [Updating DDP](#updating-ddp).

Take note that for DDP profile update to take effect ICE driver needs to be reloaded after reboot. Reboot is performed by operator after updating DDP profile to one requested in `EthernetClusterConfig`, but reloading of ICE driver is responsibility of user. Such reload can be achieved by creating systemd service that executes reload [script](../ice-driver-reload/ice-driver-reload.sh) on boot. If working on OCP a sample [MachineConfig](../ice-driver-reload/ice-driver-reload-machine-config.yaml) is provided that can be used as reference.

```shell
[Unit]
Description=ice driver reload
# Start after the network is up
Wants=network-online.target
After=network-online.target
# Also after docker.service (no effect on systems without docker)
After=docker.service
# Before kubelet.service (no effect on systems without kubernetes)
Before=kubelet.service
[Service]
Type=oneshot
TimeoutStartSec=25m
RemainAfterExit=true
ExecStart=/usr/bin/sh <path to reload script>
StandardOutput=journal+console
[Install]
WantedBy=default.target
```

### Intel Ethernet Operator - Flow Configuration

The Flow Configuration pod is a DaemonSet deployed with a CRD `FlowConfigNodeAgentDeployment` provided by Ethernet operator once it is up and running and the required DCF VF pools and their *`network attachment definitions`* are created with SRIOV Network Operator APIs. It is deployed on each node that exposes DCF VF pool as extended node resource. It is a reconcile loop which monitors the changes in each node's CR and acts on the changes. The logic implemented into this Daemon takes care of updating the cards' NIC traffic flow configuration. It consists of two components Flow Config controller container and UFT container.

#### Node Flow Configuration Controller

The Node Flow Configuration Controller watches for flow rules changes via a node specific CRD - `NodeFlowConfig` named same as the node name. Once the operator/daemon detects a change to this CR related to the Intel® E810 Flow Configuration, it tries to create/delete rules via UFT over an internal gPRC API call.

#### Unified Flow Tool

Once the Flow Config change is required the Flow Config Controller will communicate with the UFT container running a DPDK DCF application. The UFT application accepts an input with the configuration and programmes the device using a trusted VF created for this device (it is responsibility of the user to provide the trusted VFs as an allocatable K8s resource - see pre-requisites section).

### Prerequisites

The Intel Ethernet Operator has a number of prerequisites that must be met in order for complete operation of the Operator.

#### Intel Ethernet Operator - SRIOV

In order for the Flow Configuration feature to be able to configure the flow configuration of the NICs traffic the configuration must happen using a trusted Virtual Function (VF) from each Physical Function (PF) in the NIC. Usually it is the VF0 of a PF that has the trust mode set to `on` and bound to `vfio-pci` driver. This VF pool needs to be created by the user and be allocatable as a K8s resource. This VF pool will be used exclusively by the UFT container and no application container.

For user applications additional VF pools should be created separately as needed.

One way of creating and providing this trusted VF and application VFs is to configure it through SRIOV Network Operator.
In OCP environments the SRIOV Network Operator will be deployed as a dependency to Intel Ethernet Operator automatically.
The configuration and creation of the trusted VFs and application is out of scope of this Operator and is users responsibility.

#### Node Feature Discovery

To detect SRIOV capable nodes usage of Node Feature Discovery is needed. NFD detects hardware features available on each node in a Kubernetes cluster, and advertises those features using node labels and optionally node extended resources and node taints.

## Deploying the Operator

Building the operator bundle images will require Go and Operator SDK to be installed.

### Installing Go

You can install Go following the steps [here](https://go.dev/doc/install).

> Note: Intel Ethernet Operator is not compatible with Go versions below 1.19

### Installing Operator SDK

Please install Operator SDK v1.25.0 following the steps below:

```shell
export ARCH=$(case $(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(uname -m) ;; esac)
export OS=$(uname | awk '{print tolower($0)}')
export SDK_VERSION=v1.25.0
export OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/${SDK_VERSION}
curl -LO ${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH}
chmod +x operator-sdk_${OS}_${ARCH} && sudo mv operator-sdk_${OS}_${ARCH} /usr/local/bin/operator-sdk
```

Based on target cluster please follow one of the deployment steps from list below.

- [Deploy on OCP](deployment/ocp-deployment.md)
- [Deploy on Vanilla K8s](deployment/k8s-deployment.md)

### Applying custom resources

Once the operator is successfully deployed, the user interacts with it by creating CRs which will be interpreted by the operator.

Note: Example code below uses `kubectl` and the client binary. You can substitute `kubectl` with `oc` if you are operating in a OCP cluster.

#### Webserver for disconnected environment

If cluster is running in disconnected environment, then user has to create local cache (e.g webserver) which will serve required files.
Cache should be created on machine with access to Internet.
Start by creating dedicated folder for webserver.

```shell
mkdir webserver
cd webserver
```

Create nginx Dockerfile

```shell
echo "
FROM nginx
COPY files /usr/share/nginx/html
" >> Dockerfile
```

Create `files` folder

```shell
mkdir files
cd files
```

Download required packages into `files` directory

```shell
curl -OjL https://downloadmirror.intel.com/769278/E810_NVMUpdatePackage_v4_20_Linux.tar.gz
```

Build image with packages

```shell
cd ..
podman build -t webserver:1.0.0 .
```

Push image to registry that is available in disconnected environment (or copy binary image to machine via USB flash driver by using `podman save` and `podman load` commands)

```shell
podman push localhost/webserver:1.0.0 $IMAGE_REGISTRY/webserver:1.0.0
```

Create Deployment on cluster that will expose packages

```shell
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ice-cache
  namespace: default
spec:
  selector:
    matchLabels:
      run: ice-cache
  replicas: 1
  template:
    metadata:
      labels:
        run: ice-cache
    spec:
      containers:
        - name: ice-cache
          image: $IMAGE_REGISTRY/webserver:1.0.0
          ports:
            - containerPort: 80
```

And Service to make it accessible within cluster

```shell
apiVersion: v1
kind: Service
metadata:
  name: ice-cache
  namespace: default
  labels:
    run: ice-cache
spec:
  ports:
  - port: 80
    protocol: TCP
  selector:
    run: ice-cache
```

After that package will be available in cluster under following path:
```http://ice-cache.default.svc.cluster.local/E810_NVMUpdatePackage_v3_10_Linux.tar.gz```
#### Updating Firmware

To find the NIC devices belonging to the Intel® E810 NIC run following command, the user can detect the device information of the NICs from the output:

```shell
  kubectl get enc <nodename> -o jsonpath={.status}
```

To update the Firmware of the supported device run following steps:

>Note: The Physical Functions of the NICs will be updated in logical pairs. The user needs to provide the FW URL and checksum (SHA-1).

>Note: If nodeSelectors and deviceSelector are both left empty, the EthernetClusterConfig will target all compatible NICs on all available nodes.

>retryOnFail field defaults to false. If you want update to retry 5 minutes after it encounters a failure, please set to true.

Create a CR `yaml` file:

```yaml
apiVersion: ethernet.intel.com/v1
kind: EthernetClusterConfig
metadata:
  name: config
  namespace: <namespace>
spec:
  retryOnFail: true
  nodeSelectors:
    kubernetes.io/hostname: <hostname>
  deviceSelector:
    pciAddress: "<pci-address>"
  deviceConfig:
    fwURL: "<URL_to_firmware>"
    fwChecksum: "<file_checksum_SHA-1_hash>"
    fwUpdateParam: "<nvmupdate_tool_additional_params>"
```

The CR can be applied by running:

```shell
  kubectl apply -f <filename>
```

The firmware update status can be checked by running:

```shell
  kubectl get enc <nodename> -o jsonpath={.status.conditions}
```

The following status is reported:

```shell
[
  {
    "lastTransitionTime": "2021-12-17T15:25:32Z",
    "message": "Updated successfully",
    "observedGeneration": 3,
    "reason": "Succeeded",
    "status": "True",
    "type": "Updated"
  }
]
```

The user can observe the change of the cards' NICs firmware:

```shell
# kubectl get enc <nodename> -o jsonpath={.status.devices[0].firmware}
{
  "MAC": "40:a6:b7:67:1f:c0",
  "version": "3.00 0x80008271 1.2992.0"
}
```

If `fwUrl` points to external location, then you might need to configure proxy on cluster. You can configure it by using [OCP cluster-wide proxy](https://docs.openshift.com/container-platform/4.9/networking/enable-cluster-wide-proxy.html)
or by setting HTTP_PROXY, HTTPS_PROXY and NO_PROXY environmental variables in [operator's subscription](https://docs.openshift.com/container-platform/4.9/operators/admin/olm-configuring-proxy-support.html).
Be aware that operator will ignore lowercase `http_proxy` variables and will accept only uppercase variables.

#### Updating DDP

>Note: If nodeSelectors and deviceSelector are both left empty, the EthernetClusterConfig will target all compatible NICs on all available nodes

To update the DDP profile of the supported device run following steps:

Create a CR `yaml` file:

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
    ddpURL: "<URL_to_DDP>"
    ddpChecksum: "<file_checksum_SHA-1_hash>"
```

The CR can be applied by running:

```shell
  kubectl apply -f <filename>
```

Once the DDP profile update is complete, the following status is reported:

```shell
# kubectl get enc <nodename> -o jsonpath={.status.conditions}
[
  {
  "lastTransitionTime": "2021-12-17T15:25:32Z",
  "message": "Updated successfully",
  "observedGeneration": 3,
  "reason": "Succeeded",
  "status": "True",
  "type": "Updated"
  }
]
```

The user can observe the change of the cards' NICs DDP:

```shell
# kubectl get enc <nodename> -o jsonpath={.status.devices[0].DDP}|jq
{
  "packageName": "ICE COMMS Package",
  "trackId": "0xc0000002",
  "version": "1.3.30.0"
}
```

#### Deploying Flow Configuration Agent

The Flow Configuration Agent Pod runs Unified Flow Tool (UFT) to configure Flow rules for a PF. UFT requires that trust mode is enabled for the first VF (VF0) of a PF so that it has the capability of creating/modifying flow rules for that PF. This VF also needs to be bound to `vfio-pci` driver. The SRIOV VFs pools are K8s extended resources that are exposed via SRIOV Network Operator.

The VF pool consists of VF0 from all available Intel E810 series NICs PF which, in this context, we call the **Admin VF pool**. The **Admin VF pool** is associated with a NetworkAttachmentDefinition that enables these VFs trust mode 'on'. The SRIOV Network Operator can be used to create the **Admin VF pool** and the **NetworkAttachmentDefinition** needed by UFT. You can find more information on creating VF pools with SRIOV Network Operator [here](https://docs.openshift.com/container-platform/4.10/networking/hardware_networks/configuring-sriov-device.html) and creating NetworkAttachmentDefinition [here](https://docs.openshift.com/container-platform/4.10/networking/hardware_networks/configuring-sriov-net-attach.html).

The following steps will guide you through how to create the **Admin VF pool** and the **NetworkAttachmentDefinition**  needed for Flow Configuration Agent Pod.

##### Creating Trusted VF using SRIOV Network Operator

Once SRIOV Network operator is up and running we can examine the `SriovNetworkNodeStates` to view available Intel E810 Series NICs as shown below:

```shell
# kubectl get sriovnetworknodestates -n intel-ethernet-operator
NAME              AGE
worker-01   1d


# kubectl describe sriovnetworknodestates worker-01 -n intel-ethernet-operator
Name:         worker-01
Namespace:    intel-ethernet-operator
Labels:       <none>
Annotations:  <none>
API Version:  sriovnetwork.openshift.io/v1
Kind:         SriovNetworkNodeState
Metadata:
Spec:
  Dp Config Version:  42872603
Status:
  Interfaces:
    Device ID:      165f
    Driver:         tg3
    Link Speed:     100 Mb/s
    Link Type:      ETH
    Mac:            b0:7b:25:de:3f:be
    Mtu:            1500
    Name:           eno8303
    Pci Address:    0000:04:00.0
    Vendor:         14e4
    Device ID:      165f
    Driver:         tg3
    Link Speed:     -1 Mb/s
    Link Type:      ETH
    Mac:            b0:7b:25:de:3f:bf
    Mtu:            1500
    Name:           eno8403
    Pci Address:    0000:04:00.1
    Vendor:         14e4
    Device ID:      159b
    Driver:         ice
    Link Speed:     -1 Mb/s
    Link Type:      ETH
    Mac:            b4:96:91:cd:de:38
    Mtu:            1500
    Name:           eno12399
    Pci Address:    0000:31:00.0
    Vendor:         8086
    Device ID:      159b
    Driver:         ice
    Link Speed:     -1 Mb/s
    Link Type:      ETH
    Mac:            b4:96:91:cd:de:39
    Mtu:            1500
    Name:           eno12409
    Pci Address:    0000:31:00.1
    Vendor:         8086
    Device ID:      1592
    Driver:         ice
    E Switch Mode:  legacy
    Link Speed:     -1 Mb/s
    Link Type:      ETH
    Mac:            b4:96:91:aa:d8:40
    Mtu:            1500
    Name:           ens1f0
    Pci Address:    0000:18:00.0
    Totalvfs:       128
    Vendor:         8086
    Device ID:      1592
    Driver:         ice
    E Switch Mode:  legacy
    Link Speed:     -1 Mb/s
    Link Type:      ETH
    Mac:            b4:96:91:aa:d8:41
    Mtu:            1500
    Name:           ens1f1
    Pci Address:    0000:18:00.1
    Totalvfs:       128
    Vendor:         8086
  Sync Status:      Succeeded
Events:             <none>

```

By looking at the sriovnetworknodestates status we can find the NIC information such as PCI address and Interface names to define `SriovNetworkNodePolicy` to create required VF pools.

For example, the following three `SriovNetworkNodePolicy` CRs will create a trusted VF pool name with resourceName `cvl_uft_admin` along with two additional VF pools for application.

> Please note that, the "uft-admin-policy" SriovNetworkNodePolicy below uses `pfNames:` with VF index range selectors to target VF0 only of Intel E810 series NIC. More information on using VF partitioning can be found [here](https://docs.openshift.com/container-platform/4.10/networking/hardware_networks/configuring-sriov-device.html#nw-sriov-nic-partitioning_configuring-sriov-device).

```yaml
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetworkNodePolicy
metadata:
  name: uft-admin-policy
  namespace: intel-ethernet-operator
spec:
  deviceType: vfio-pci
  nicSelector:
    pfNames:
    - ens1f0#0-0
    - ens1f1#0-0
    vendor: "8086"
  nodeSelector:
    feature.node.kubernetes.io/network-sriov.capable: 'true'
  numVfs: 8
  priority: 99
  resourceName: cvl_uft_admin
---
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetworkNodePolicy
metadata:
  name: cvl-vfio-policy
  namespace: intel-ethernet-operator
spec:
  deviceType: vfio-pci
  nicSelector:
    pfNames:
    - ens1f0#1-3
    - ens1f1#1-3
    vendor: "8086"
  nodeSelector:
    feature.node.kubernetes.io/network-sriov.capable: 'true'
  numVfs: 8
  priority: 89
  resourceName: cvl_vfio
---
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetworkNodePolicy
metadata:
  name: cvl-iavf-policy
  namespace: intel-ethernet-operator
spec:
  deviceType: netdevice
  nicSelector:
    pfNames:
    - ens1f0#4-7
    - ens1f1#4-7
    vendor: "8086"
  nodeSelector:
    feature.node.kubernetes.io/network-sriov.capable: 'true'
  numVfs: 8
  priority: 79
  resourceName: cvl_iavf

```

Save the above yaml in file name `sriov-network-policy.yaml` and then apply this to create the VF pools.

The CR can be applied by running:

```shell
  kubectl create -f sriov-network-policy.yaml
```

##### Check node status

Check node status to confirm that cvl_uft_admin resource pool registered DCF capable VFs of the node

```shell
# kubectl describe node worker-01 -n intel-ethernet-operator | grep -i allocatable -A 20
Allocatable:
  bridge.network.kubevirt.io/cni-podman0:  1k
  cpu:                                     108
  devices.kubevirt.io/kvm:                 1k
  devices.kubevirt.io/tun:                 1k
  devices.kubevirt.io/vhost-net:           1k
  ephemeral-storage:                       468315972Ki
  hugepages-1Gi:                           0
  hugepages-2Mi:                           8Gi
  memory:                                  518146752Ki
  openshift.io/cvl_iavf:                   8
  openshift.io/cvl_uft_admin:              2
  openshift.io/cvl_vfio:                   6
  pods:                                    250
```

##### Create DCF capable SRIOV Network

Next, we will need to create SRIOV network attachment definition for the DCF VF pool as shown below:

```shell
cat <<EOF | kubectl apply -f -
apiVersion: sriovnetwork.openshift.io/v1
kind: SriovNetwork
metadata:
  name: sriov-cvl-dcf
spec:
  trust: 'on'
  networkNamespace: intel-ethernet-operator
  resourceName: cvl_uft_admin
EOF
```

Note if the above does not successfully set trust mode to on for vf 0, you can do it manually using this command:

```shell
  ip link set <PF_NAME> vf 0 trust on
```

##### Build UFT image

```shell
  export IMAGE_REGISTRY=<OCP Image registry>

  git clone https://github.com/intel/UFT.git

  git checkout v22.07

  make dcf-image

  docker tag uft:v22.07 $IMAGE_REGISTRY/uft:v22.07

  docker push $IMAGE_REGISTRY/uft:v22.07
```

**NOTE:** If you are using a version of [sriov-network-device-plugin](https://github.com/k8snetworkplumbingwg/sriov-network-device-plugin) newer than v3.5.1, you will need to apply the "``patches/uft-fix.patch``" file from the IEO repository on "``images/entrypoint.sh``" located in the UFT repository before building the uft image (running ``make dcf-image``).

```shell
patch -u <UFT_repo>/images/entrypoint.sh < <IEO_repo>/patches/uft-fix.patch
```

##### Creating FlowConfig Node Agent Deployment CR

> Note: The Admin VF pool prefix in `DCFVfPoolName` should match how it is shown on node description in [Check node status](#check-node-status) section.

```shell
cat <<EOF | kubectl apply -f -
apiVersion: flowconfig.intel.com/v1
kind: FlowConfigNodeAgentDeployment
metadata:
  labels:
    control-plane: flowconfig-daemon
  name: flowconfig-daemon-deployment
  namespace: intel-ethernet-operator
spec:
  DCFVfPoolName: openshift.io/cvl_uft_admin
  NADAnnotation: sriov-cvl-dcf
EOF
```

##### Verifying that FlowConfig Daemon is running on available nodes:

```shell
# kubectl get pods -n intel-ethernet-operator
NAME                                                          READY   STATUS    RESTARTS   AGE
clv-discovery-kwjkb                                           1/1     Running   0          44h
clv-discovery-tpqzb                                           1/1     Running   0          44h
flowconfig-daemon-worker-01                                   2/2     Running   0          44h
fwddp-daemon-m8d4w                                            1/1     Running   0          44h
intel-ethernet-operator-controller-manager-79c4d5bf6d-bjlr5   1/1     Running   0          44h
intel-ethernet-operator-controller-manager-79c4d5bf6d-txj5q   1/1     Running   0          44h

# kubectl logs -n intel-ethernet-operator flowconfig-daemon-worker-01 -c uft
Generating server_conf.yaml file...
Done!
server :
    ld_lib : "/usr/local/lib64"
ports_info :
    - pci  : "0000:18:01.0"
      mode : dcf
do eal init ...
[{'pci': '0000:18:01.0', 'mode': 'dcf'}]
[{'pci': '0000:18:01.0', 'mode': 'dcf'}]
the dcf cmd line is: a.out -c 0x30 -n 4 -a 0000:18:01.0,cap=dcf -d /usr/local/lib64 --file-prefix=dcf --
EAL: Detected 96 lcore(s)
EAL: Detected 2 NUMA nodes
EAL: Detected shared linkage of DPDK
EAL: Multi-process socket /var/run/dpdk/dcf/mp_socket
EAL: Selected IOVA mode 'VA'
EAL: No available 1048576 kB hugepages reported
EAL: VFIO support initialized
EAL: Using IOMMU type 1 (Type 1)
EAL: Probe PCI driver: net_iavf (8086:1889) device: 0000:18:01.0 (socket 0)
EAL: Releasing PCI mapped resource for 0000:18:01.0
EAL: Calling pci_unmap_resource for 0000:18:01.0 at 0x2101000000
EAL: Calling pci_unmap_resource for 0000:18:01.0 at 0x2101020000
EAL: Using IOMMU type 1 (Type 1)
EAL: Probe PCI driver: net_ice_dcf (8086:1889) device: 0000:18:01.0 (socket 0)
ice_load_pkg_type(): Active package is: 1.3.30.0, ICE COMMS Package (double VLAN mode)
TELEMETRY: No legacy callbacks, legacy socket not created
grpc server start ...
now in server cycle
```

##### Creating Flow Configuration rules with Intel Ethernet Operator

With trusted VFs and application VFs ready to be configured, program the Flow Configuration by running:

Create a sample Node specific NodeFlowConfig CR named same as a target node with empty spec:

```shell
cat <<EOF | kubectl apply -f -
apiVersion: flowconfig.intel.com/v1
kind: NodeFlowConfig
metadata:
  name: worker-01
spec:
EOF
```


Check status of CR:

```shell
# kubectl describe nodeflowconfig worker-01

Name:         worker-01
Namespace:    intel-ethernet-operator
Labels:       <none>
Annotations:  <none>
API Version:  flowconfig.intel.com/v1
Kind:         NodeFlowConfig
Metadata:
Status:
  Port Info:
    Port Id:    0
    Port Mode:  dcf
    Port Pci:   0000:18:01.0
Events:         <none>

```
You can see the DCF port information from NodeFlowConfig CR status for a node. These port information can be used to identify for which port on a node the Flow rules should be applied.


##### Update a sample Node Flow Configuration rule

Please see the [NodeFlowConfig Spec](flowconfig-daemon/creating-rules.md) for detailed specification of supported rules.
We can update the Node Flow configuration with a sample rule for a target port as shown below:

```shell
cat <<EOF | kubectl apply -f -
apiVersion: flowconfig.intel.com/v1
kind: NodeFlowConfig
metadata:
  name: worker-01
  namespace: intel-ethernet-operator
spec:
  rules:
    - pattern:
        - type: RTE_FLOW_ITEM_TYPE_ETH
        - type: RTE_FLOW_ITEM_TYPE_IPV4
          spec:
            hdr:
              src_addr: 10.56.217.9
          mask:
            hdr:
              src_addr: 255.255.255.255
        - type: RTE_FLOW_ITEM_TYPE_END
      action:
        - type: RTE_FLOW_ACTION_TYPE_DROP
        - type: RTE_FLOW_ACTION_TYPE_END
      portId: 0
      attr:
        ingress: 1
EOF
```

Validate that Flow Rules are applied by the controller from UFT logs.

```shell
kubectl logs flowconfig-daemon-worker uft
Generating server_conf.yaml file...
Done!
server :
    ld_lib : "/usr/local/lib64"
ports_info :
    - pci  : "0000:18:01.0"
      mode : dcf
do eal init ...
[{'pci': '0000:18:01.0', 'mode': 'dcf'}]
[{'pci': '0000:18:01.0', 'mode': 'dcf'}]
the dcf cmd line is: a.out -c 0x30 -n 4 -a 0000:18:01.0,cap=dcf -d /usr/local/lib64 --file-prefix=dcf --
EAL: Detected 96 lcore(s)
EAL: Detected 2 NUMA nodes
EAL: Detected shared linkage of DPDK
EAL: Multi-process socket /var/run/dpdk/dcf/mp_socket
EAL: Selected IOVA mode 'VA'
EAL: No available 1048576 kB hugepages reported
EAL: VFIO support initialized
EAL: Using IOMMU type 1 (Type 1)
EAL: Probe PCI driver: net_iavf (8086:1889) device: 0000:18:01.0 (socket 0)
EAL: Releasing PCI mapped resource for 0000:18:01.0
EAL: Calling pci_unmap_resource for 0000:18:01.0 at 0x2101000000
EAL: Calling pci_unmap_resource for 0000:18:01.0 at 0x2101020000
EAL: Using IOMMU type 1 (Type 1)
EAL: Probe PCI driver: net_ice_dcf (8086:1889) device: 0000:18:01.0 (socket 0)
ice_load_pkg_type(): Active package is: 1.3.30.0, ICE COMMS Package (double VLAN mode)
TELEMETRY: No legacy callbacks, legacy socket not created
grpc server start ...
now in server cycle
flow.rte_flow_attr
flow.rte_flow_item
flow.rte_flow_item
flow.rte_flow_item_ipv4
flow.rte_ipv4_hdr
flow.rte_flow_item_ipv4
flow.rte_ipv4_hdr
flow.rte_flow_item
flow.rte_flow_action
flow.rte_flow_action
rte_flow_attr(group=0, priority=0, ingress=1, egress=0, transfer=0, reserved=0) [rte_flow_item(type_=9, spec=None, last=None, mask=None), rte_flow_item(type_=11, spec=rte_flow_item_ipv4(hdr=rte_ipv4_hdr(version_ihl=0, type_of_service=0, total_length=0, packet_id=0, fragment_offset=0, time_to_live=0, next_proto_id=0, hdr_checksum=0, src_addr=171497737, dst_addr=0)), last=None, mask=rte_flow_item_ipv4(hdr=rte_ipv4_hdr(version_ihl=0, type_of_service=0, total_length=0, packet_id=0, fragment_offset=0, time_to_live=0, next_proto_id=0, hdr_checksum=0, src_addr=4294967295, dst_addr=0))), rte_flow_item(type_=0, spec=None, last=None, mask=None)] [rte_flow_action(type_=7, conf=None), rte_flow_action(type_=0, conf=None)]
rte_flow_attr(group=0, priority=0, ingress=1, egress=0, transfer=0, reserved=0)
1
Finish ipv4: {'hdr': {'version_ihl': 0, 'type_of_service': 0, 'total_length': 0, 'packet_id': 0, 'fragment_offset': 0, 'time_to_live': 0, 'next_proto_id': 0, 'hdr_checksum': 0, 'src_addr': 165230602, 'dst_addr': 0}}
Finish ipv4: {'hdr': {'version_ihl': 0, 'type_of_service': 0, 'total_length': 0, 'packet_id': 0, 'fragment_offset': 0, 'time_to_live': 0, 'next_proto_id': 0, 'hdr_checksum': 0, 'src_addr': 4294967295, 'dst_addr': 0}}
rte_flow_action(type_=7, conf=None)
rte_flow_action(type_=0, conf=None)
Validate ok...
flow.rte_flow_attr
flow.rte_flow_item
flow.rte_flow_item
flow.rte_flow_item_ipv4
flow.rte_ipv4_hdr
flow.rte_flow_item_ipv4
flow.rte_ipv4_hdr
flow.rte_flow_item
flow.rte_flow_action
flow.rte_flow_action
rte_flow_attr(group=0, priority=0, ingress=1, egress=0, transfer=0, reserved=0) [rte_flow_item(type_=9, spec=None, last=None, mask=None), rte_flow_item(type_=11, spec=rte_flow_item_ipv4(hdr=rte_ipv4_hdr(version_ihl=0, type_of_service=0, total_length=0, packet_id=0, fragment_offset=0, time_to_live=0, next_proto_id=0, hdr_checksum=0, src_addr=171497737, dst_addr=0)), last=None, mask=rte_flow_item_ipv4(hdr=rte_ipv4_hdr(version_ihl=0, type_of_service=0, total_length=0, packet_id=0, fragment_offset=0, time_to_live=0, next_proto_id=0, hdr_checksum=0, src_addr=4294967295, dst_addr=0))), rte_flow_item(type_=0, spec=None, last=None, mask=None)] [rte_flow_action(type_=7, conf=None), rte_flow_action(type_=0, conf=None)]
rte_flow_attr(group=0, priority=0, ingress=1, egress=0, transfer=0, reserved=0)
rte_flow_attr(group=0, priority=0, ingress=1, egress=0, transfer=0, reserved=0)
1
Finish ipv4: {'hdr': {'version_ihl': 0, 'type_of_service': 0, 'total_length': 0, 'packet_id': 0, 'fragment_offset': 0, 'time_to_live': 0, 'next_proto_id': 0, 'hdr_checksum': 0, 'src_addr': 165230602, 'dst_addr': 0}}
Finish ipv4: {'hdr': {'version_ihl': 0, 'type_of_service': 0, 'total_length': 0, 'packet_id': 0, 'fragment_offset': 0, 'time_to_live': 0, 'next_proto_id': 0, 'hdr_checksum': 0, 'src_addr': 4294967295, 'dst_addr': 0}}
rte_flow_action(type_=7, conf=None)
rte_flow_action(type_=0, conf=None)
free attr
free item ipv4
free item ipv4
free list item
free list action
Flow rule #0 created on port 0
```

## Hardware Validation Environment

- Intel® Ethernet Network Adapter E810-XXVDA2
- 3nd Generation Intel® Xeon® processor platform

## Summary

The Intel Ethernet Operator is a functional tool to manage the update of Intel® E810 NICs FW and DDP profile, as well as the programming of the NICs VFs Flow Configuration autonomously in a Cloud Native OpenShift environment based on user input. It is easy in use by providing simple steps to apply the Custom Resources to configure various aspects of the device.
