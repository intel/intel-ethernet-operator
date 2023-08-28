```text
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2020-2023 Intel Corporation
```

# Deploy Intel Ethernet Operator on OCP cluster

## Technical Requirements and Dependencies

The Intel Ethernet Operator on OCP has the following requirements:

- Intel® Ethernet Network Adapter E810
- OpenShift 4.9 or newer
- Out of tree ICE driver
- [Intel® Network Adapter Driver for E810 Series Devices](https://www.intel.com/content/www/us/en/download/19630/intel-network-adapter-driver-for-e810-series-devices-under-linux.html)
- IOMMU enabled
- Hugepage memory configured
- Node Feature Discovery Operator with basic NFD CR applied
- SRIOV Network Operator deployed
- External Docker Registry is setup and Cluster is configured to use that

### Intel Ethernet Operator - OOT ICE Driver Update

In order for the FW update and Flow Configuration to be possible the platform needs to provide an [OOT ICE driver](https://www.intel.com/content/www/us/en/download/19630/intel-network-adapter-driver-for-e810-series-devices-under-linux.html). This is required since current implementations of in-tree drivers do not support all required features.
It is a responsibility of the cluster admin to provide and install this driver and it is out of scope of this Operator at this time. See the [kmm-ice-install-ocp document](oot-ice-driver/kmm-ice-install-ocp.md) for sample instructions on how to install the driver using KMMO.

## Deploying the Operator

The Intel Ethernet Operator can be deployed by building the Bundle image and the Operator images from source. An external registry is necessary to push the images during build.

### Installing the Dependencies

Before building and installing the Operator, provide and install the OOT Intel ICE driver to the platforms. The driver can be downloaded from [Intel Download Centre](https://www.intel.com/content/www/us/en/download/19630/intel-network-adapter-driver-for-e810-series-devices-under-linux.html).

On OCP deployments, the SRIOV Network operator will be deployed automatically as a dependency to the Intel Ethernet Operator.

### Building the Operator from Source

To build the Operator the images must be built from source, in order to build execute the following steps:

> Note: The arguments are to be replaced with the following:
>
- VERSION is the version to be applied to the bundle ie. `0.0.1`.
- IMAGE_REGISTRY is the address of the registry where the build images are to be pushed to ie. `my.private.registry.com`.
- TLS_VERIFY defines whether connection to registry need TLS verification, default is `false`.
- TARGET_PLATFORM specific platform for which operator will be built. Supported values are `OCP` and `K8S`. If operator is built for other platform than `OCP`,
then user has to manually install sriov-network-operator as described [on sriov-network-operator page](https://github.com/k8snetworkplumbingwg/sriov-network-operator). Default is `OCP`

```shell
$ cd intel-ethernet-operator
$ make VERSION=$(VERSION) IMAGE_REGISTRY=$(IMAGE_REGISTRY) TLS_VERIFY=$(TLS_VERIFY) TARGET_PLATFORM=$(TARGET_PLATFORM) build_all push_all catalog-build catalog-push
```

### Installing the Bundle

Once the operator images are built and accessible inside the cluster, the operator is to be installed by running the following:

Create a namespace for the operator:  

```shell
$ oc create ns intel-ethernet-operator
```

Create the following `Catalog Source` `yaml` file:

> Note: The REGISTRY_ADDRESS and VERSION need to be replaced:
>
> - VERSION is the version to be applied to the bundle ie. `0.0.2`.
> - IMAGE_REGISTRY is the address of the registry where the build images are to be pushed to ie. `my.private.registry.com`.

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: intel-ethernet-operators
  namespace: openshift-marketplace
spec:
  sourceType: grpc
  image: <IMAGE_REGISTRY>/intel-ethernet-operator-catalog:<VERSION>
  publisher: Intel
  displayName: Intel ethernet operators(Local)
```

Create the `Catalog Source`

```shell
$ oc apply -f <filename>
```

Create the following `yaml` files including `Subscription` and `OperatorGroup`:

```yaml
---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: intel-ethernet-operator
  namespace: intel-ethernet-operator
spec:
  targetNamespaces:
    - intel-ethernet-operator

---

apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: intel-ethernet-subscription
  namespace: intel-ethernet-operator
spec:
  channel: alpha
  name: intel-ethernet-operator
  source: intel-ethernet-operators
  sourceNamespace: openshift-marketplace
```

Subscribe to and install the operator:

```shell
$ oc apply -f <filename>
```

Check that the operator is deployed:
> Note: SRIOV Network Operator pods deployed as a dependency in OCP environments.

```text
$ oc get pods -n intel-ethernet-operator
NAME                                                          READY   STATUS    RESTARTS      AGE
clv-discovery-db6j7                                           1/1     Running   0             23h
clv-discovery-fl5n6                                           1/1     Running   0             23h
clv-discovery-pqhtl                                           1/1     Running   0             23h
fwddp-daemon-4cmn7                                            1/1     Running   0             23h
fwddp-daemon-5jjzw                                            1/1     Running   0             23h
intel-ethernet-operator-controller-manager-75d4449bfb-cx65b   1/1     Running   0             23h
intel-ethernet-operator-controller-manager-75d4449bfb-dhqv5   1/1     Running   0             23h
network-resources-injector-g27j2                              1/1     Running   0             23h
network-resources-injector-kddh4                              1/1     Running   0             23h
network-resources-injector-vqhqk                              1/1     Running   0             23h
operator-webhook-5gbz8                                        1/1     Running   0             23h
operator-webhook-c42n6                                        1/1     Running   0             23h
operator-webhook-rtt7v                                        1/1     Running   0             23h
sriov-network-config-daemon-6xdlg                             3/3     Running   0             23h
sriov-network-config-daemon-gp9xz                             3/3     Running   0             23h
sriov-network-config-daemon-sqgck                             3/3     Running   0             23h
sriov-network-operator-78cf54b79d-ll9nz                       1/1     Running   0             45h
```
