```text
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2020-2021 Intel Corporation
```

# Intel Ethernet Operator

## FlowConfig Daemon Controller

> **Disclaimer**: This Operator is still a _proof-of-concept_ and only intended to be deployed in an isolated lab environment for testing. It is not intended to be deployed in any production capacity.

The FlowConfig Daemon is K8s custom controller that runs as node level agent on each node.
This Operator mediator between cluster admin and the underlying DCF tool to manage Intel Columbiaville(800 series) NICs to configure and manage advanced switch filter and ACL rules.

This Operator is built on using Operator SDK v1.7.2.

```
+-------------------------------------------+
|       FlowConfig Daemon Controller        |
+---------------------+---------------------+
                      |
                      |gRPC(localhost)
                      |
                      |
+---------------------+---------------------+
|                    DCF                    |
+---------------------+---------------------+
                      |
                      |
                      |
                +-----+-----+
                |    VF     |
                | trust on  |
+---------------+-----------+---------------+
|             Intel E810 NIC                |
+-------------------------------------------+
```
Fig: FlowConfig Daemon Controller relationship diagram with DCF and CVL NIC.


### Dependencies
- A K8s baremetal cluster configured with Multus CNI
- Docker
- SR-IOV device plugin
- SR-IOV CNI
- Multus
- Hugepage support
- Golang
- Operator SDK: v1.7.2
- Intel E810 Firmware version: 2.22
- Intel ICE Driver version: 1.2.1
- K8s cert-manager

#### K8s cert-manager
The Ethernet Operator deploys a validation webhook for CRD object validation. This webhook is exposed as K8s service and requires self-signed certificates. The Operator SDK uses [K8s cert-manager](https://cert-manager.io/docs/)'s cert-injection mechanism for webhook manifests.
For this reason the cert-manager must be installed in the cluster prior to the operator deployment. To install Cert-manager please refer to its [installation guide](https://cert-manager.io/docs/installation/kubernetes/).
### Environment setup
For DCF to work, VT-d needs to be enabled in system BIOS and the iommu needs to be turned on in Kernel parameters.

### Prepare NIC
#### Check FW and Drivers
Update NIC's firmware with latest version and then install latest version of ICE driver. Please look for latest drivers on [official page](https://www.intel.com/content/www/us/en/products/details/ethernet/800-network-adapters/e810-network-adapters/docs.html?s=Newest).

#### Create SR-IOV Virtual functions
```
echo 4 > /sys/class/net/ens785f0/device/sriov_numvfs
```

Create VFs for each PF you intend to use

Confirm that VFs are created successfully:
```
# ip link show ens785f0
20: ens785f0: <NO-CARRIER,BROADCAST,MULTICAST,UP> mtu 1500 qdisc mq state DOWN mode DEFAULT group default qlen 1000
    link/ether 68:05:ca:a3:71:b8 brd ff:ff:ff:ff:ff:ff
    vf 0     link/ether 4a:fd:1c:54:f9:b9 brd ff:ff:ff:ff:ff:ff, spoof checking on, link-state auto, trust off
    vf 1     link/ether c6:d8:55:6a:9e:85 brd ff:ff:ff:ff:ff:ff, spoof checking on, link-state auto, trust off
    vf 2     link/ether 7e:85:44:46:75:86 brd ff:ff:ff:ff:ff:ff, spoof checking on, link-state auto, trust off
    vf 3     link/ether 42:57:11:f1:51:93 brd ff:ff:ff:ff:ff:ff, spoof checking on, link-state auto, trust off
```

#### Bind vfio-pci
With DPDK [devbind script](https://github.com/DPDK/dpdk/blob/main/usertools/dpdk-devbind.py) bind vfio-pci driver to the first VF on each PF you wish to use

```
# dpdk-devbind.py -s

Network devices using DPDK-compatible driver
============================================


Network devices using kernel driver
===================================
0000:17:00.0 'Ethernet Controller E810-C for QSFP 1592' if=ens785f0 drv=ice unused=vfio-pci
0000:17:00.1 'Ethernet Controller E810-C for QSFP 1592' if=ens785f1 drv=ice unused=vfio-pci
0000:17:01.0 'Ethernet Adaptive Virtual Function 1889' if= drv=iavf unused=vfio-pci
0000:17:01.1 'Ethernet Adaptive Virtual Function 1889' if= drv=iavf unused=vfio-pci
0000:17:01.2 'Ethernet Adaptive Virtual Function 1889' if= drv=iavf unused=vfio-pci
0000:17:01.3 'Ethernet Adaptive Virtual Function 1889' if=ens785f0v3 drv=iavf unused=vfio-pci
0000:67:00.0 'Ethernet Connection X722 for 10GBASE-T 37d2' if=eno1 drv=i40e unused=vfio-pci *Active*
0000:67:00.1 'Ethernet Connection X722 for 10GBASE-T 37d2' if=eno2 drv=i40e unused=vfio-pci

```

Bind each VF with vfio-pci driver using it's PCI address:
```
# dpdk-devbind.py -b vfio-pci 0000:17:01.0
```

Confirm that vfio-pci driver bind was successful:
```
# dpdk-devbind.py -s

Network devices using DPDK-compatible driver
============================================
0000:17:01.0 'Ethernet Adaptive Virtual Function 1889' drv=vfio-pci unused=iavf

Network devices using kernel driver
===================================
0000:17:00.0 'Ethernet Controller E810-C for QSFP 1592' if=ens785f0 drv=ice unused=vfio-pci
0000:17:00.1 'Ethernet Controller E810-C for QSFP 1592' if=ens785f1 drv=ice unused=vfio-pci
0000:17:01.1 'Ethernet Adaptive Virtual Function 1889' if= drv=iavf unused=vfio-pci
0000:17:01.2 'Ethernet Adaptive Virtual Function 1889' if= drv=iavf unused=vfio-pci
0000:17:01.3 'Ethernet Adaptive Virtual Function 1889' if=ens785f0v3 drv=iavf unused=vfio-pci
0000:67:00.0 'Ethernet Connection X722 for 10GBASE-T 37d2' if=eno1 drv=i40e unused=vfio-pci *Active*
0000:67:00.1 'Ethernet Connection X722 for 10GBASE-T 37d2' if=eno2 drv=i40e unused=vfio-pci

```
#### Build docker image for DCF
Clone DCF Tool repo from gitlab
```
# git clone ssh://git@gitlab.devtools.intel.com:29418/zhaoyanc/dcf-tool.git

# cd dcf-tool

# git checkout dev_21.08 [WILL CHANGE WHEN UFT TOOL RELEASE]
```

Assuming you have the latest version of docker installed in your node.

Build dcf docker image:
```
make dcf-image
```
Docker needs to be able to download dependencies from Internet. If required you can add proxy information as follows(assuming http_proxy and https_proxy env variables are set correctly):

```
make dcf-image HTTP_PROXY=$http_proxy HTTPS_PROXY=$https_proxy
```


#### Deploy SR-IOV device plugin and SR-IOV CNI

##### Deploy SR-IOV DP
Move back to Intel Ethernet Operator repo
Update the "pfNames" parameters in `config/samples/k8s/configMap.yaml` file to match with #0 of each CVL PF

NOTE: pfNames should include all vfs you intend to use on all nodes

###### Update configMap:

```
kubectl apply -f config/samples/k8s/configMap.yaml
```

###### Deploy SR-IOV Device plugin DaemonSet:

```
kubectl apply -f config/samples/k8s/sriovdp-daemonset.yaml
```

###### Install SR-IOV CNI

Follow [SR-IOV CNI documentation](https://github.com/k8snetworkplumbingwg/sriov-cni#kubernetes-quick-start) to install it

#### Create Hugepages
DCF is a DPDK application and requires hugepage memory.
See [DPDK system requirements](http://doc.dpdk.org/guides/linux_gsg/sys_reqs.html) for hugepage memory.

To create 2M hugepage memory on a mulit-socket system you can run following commands:
```
echo 1024 > /sys/devices/system/node/node0/hugepages/hugepages-2048kB/nr_hugepages
echo 1024 > /sys/devices/system/node/node1/hugepages/hugepages-2048kB/nr_hugepages
```
The above command will create 2GB hugepages on each NUMA zone 4GB in total.

Verify that hugepage is created:
```
# cat /proc/meminfo | grep -i huge
AnonHugePages:    716800 kB
ShmemHugePages:        0 kB
HugePages_Total:    4096
HugePages_Free:     4096
HugePages_Rsvd:        0
HugePages_Surp:        0
Hugepagesize:       2048 kB
Hugetlb:         8388608 kB
```

###  Install Operator SDK
Install Operator SDK v1.7.2 using installation instructions from [here](https://sdk.operatorframework.io/docs/installation/install-operator-sdk/).

### Build Intel Ethernet Operator
```
git clone https://github.com/otcshare/intel-ethernet-operator.git
cd intel-ethernet-operator
```

##### Update DCFVfPoolName and NADAnnotation
In "config/flowconfig-daemon/add_uft_container.yaml: Update DCFVfPoolName to match your resource pool as defined in the configMap and NADAnnotaion to match your NetworkAttachmentDefinition

```
git checkout [TO-DO]
make docker-build HTTP_PROXY=$http_proxy HTTPS_PROXY=$https_proxy
```
### Build FlowConfig Dameon
```
make docker-build-flowconfig HTTP_PROXY=$http_proxy HTTPS_PROXY=$https_proxy
```
### Deploy Intel Ethernet Operator
```
make deploy
```
### Verify deployment
The below deployment is set up on a 2 node cluster

```
kubectl get all -n intel-ethernet-operator-system
NAME                                                              READY   STATUS    RESTARTS   AGE
pod/flowconfig-daemon-silpixa00399883                             2/2     Running   0          21s
pod/flowconfig-daemon-silpixa00401200c                            2/2     Running   0          21s
pod/intel-ethernet-operator-controller-manager-5b68bb54fd-274t5   2/2     Running   0          29s

NAME                                                                 TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
service/intel-ethernet-operator-controller-manager-metrics-service   ClusterIP   10.109.6.109   <none>        8443/TCP   29s
service/intel-ethernet-operator-webhook-service                      ClusterIP   10.111.55.32   <none>        443/TCP    29s

NAME                                                         READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/intel-ethernet-operator-controller-manager   1/1     1            1           29s

NAME                                                                    DESIRED   CURRENT   READY   AGE
replicaset.apps/intel-ethernet-operator-controller-manager-5b68bb54fd   1         1         1       29s

```

### Verify UFT is running
For each 'flowconfig-daemon-[NODE_NAME]' pod, you can verfiy UFT is running correctly by getting the logs

```
kubectl -n intel-ethernet-operator-system logs flowconfig-daemon-silpixa00401200c -c uft
Generating server_conf.yaml file...
Done!
server :
    ld_lib : "/usr/local/lib64"
ports_info :
    - pci  : "0000:5e:01.0"
      mode : dcf
do eal init ...
[{'pci': '0000:5e:01.0', 'mode': 'dcf'}]
[{'pci': '0000:5e:01.0', 'mode': 'dcf'}]
the dcf cmd line is: a.out -c 0x30 -n 4 -a 0000:5e:01.0,cap=dcf -d /usr/local/lib64 --file-prefix=dcf --
EAL: Detected 88 lcore(s)
EAL: Detected 2 NUMA nodes
EAL: Detected shared linkage of DPDK
EAL: Multi-process socket /var/run/dpdk/dcf/mp_socket
EAL: Selected IOVA mode 'VA'
EAL: No available 1048576 kB hugepages reported
EAL: VFIO support initialized
EAL: Using IOMMU type 1 (Type 1)
EAL: Probe PCI driver: net_iavf (8086:1889) device: 0000:5e:01.0 (socket 0)
EAL: Releasing PCI mapped resource for 0000:5e:01.0
EAL: Calling pci_unmap_resource for 0000:5e:01.0 at 0x2101000000
EAL: Calling pci_unmap_resource for 0000:5e:01.0 at 0x2101020000
EAL: Using IOMMU type 1 (Type 1)
EAL: Probe PCI driver: net_ice_dcf (8086:1889) device: 0000:5e:01.0 (socket 0)
ice_load_pkg_type(): Active package is: 1.3.26.0, ICE OS Default Package (double VLAN mode)
TELEMETRY: No legacy callbacks, legacy socket not created
grpc server start ...
now in server cycle
```
### Testing
#### Create sample ACL rules:
Change the `name` value in sample config file config/samples/flowconfig_v1_test.yaml to the node name of target node.

```
kubectl apply -f config/samples/flowconfig_v1_test.yaml
```

Check Ethernet Operator logs to verify that the ACL rules are created:

```
kubectl -n intel-ethernet-operator-system logs intel-ethernet-operator-controller-manager-5b68bb54fd-cgfz4 -c manager

...
2021-09-15T03:22:53.997Z        DEBUG   controller-runtime.webhook.webhooks     received request        {"webhook": "/validate-flowconfig-intel-com-v1-nodeflowconfig", "UID": "5eaaf3e2-e0ae-4b3f-adc7-1cdb1958d851", "kind": "flowconfig.intel.com/v1, Kind=NodeFlowConfig", "resource": {"group":"flowconfig.intel.com","version":"v1","resource":"nodeflowconfigs"}}
2021-09-15T03:22:54.095Z        INFO    nodeflowconfig-resource validate create {"name": "silpixa00399883"}
2021-09-15T03:22:54.097Z        DEBUG   controller-runtime.webhook.webhooks     wrote response  {"webhook": "/validate-flowconfig-intel-com-v1-nodeflowconfig", "code": 200, "reason": "", "UID": "5eaaf3e2-e0ae-4b3f-adc7-1cdb1958d851", "allowed": true}
...
```

To read more about creating ACL rules see this [creating-rules.md](docs/flowconfig-daemon/creating-rules.md) user guide. Note that some items may require a recent [Ice COMMS DDP package](https://downloadcenter.intel.com/download/29889/Intel-Ethernet-800-Series-Telecommunication-Comms-Dynamic-Device-Personalization-DDP-Package) to be loaded.

### Clean up
To clean up and teardown the Ethernet operator run:
```
make undeploy
```
