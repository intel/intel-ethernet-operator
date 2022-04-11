```text
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2022 Intel Corporation
```

# Deploy Intel Ethernet Operator on Vanilla K8s with Intel CEK

## Technical Requirements and Dependencies

The Intel Ethernet Operator on vanilla K8s has the following requirements:

- Intel® Ethernet Network Adapter E810
- Vanilla Kubernetes cluster using [Intel Container Experience Kits](https://github.com/intel/container-experience-kits) v21.08 or later with `remote_fp` profile.
- [Intel® Network Adapter Driver for E810 Series Devices](https://www.intel.com/content/www/us/en/download/19630/intel-network-adapter-driver-for-e810-series-devices-under-linux.html)
- IOMMU enabled
- Hugepage memory configured
- Node Feature Discovery Operator with basic NFD CR applied
- SRIOV Network Operator deployed
- External Docker Registry is setup and Cluster is configured to use that
- Operator Lifecycle Manager deployed


## Deploying Operator Lifecycle Manager (OLM) Operator

The following will install OLM v0.20.0 in your cluster.

```
# kubectl create -f  https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/v0.20.0/deploy/upstream/quickstart/crds.yaml
# kubectl create -f  https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/v0.20.0/deploy/upstream/quickstart/olm.yaml
```

## Deploying the Operator

The Intel Ethernet Operator can be deployed by building the Bundle image and the Operator images from source. An external registry is necessary to push the images during build.


### Building the Operator from Source

To build the Operator, the images must be built from source. In order to build, execute the following steps:

> Note: The arguments are to be replaced with the following:
>
* VERSION is the version to be applied to the bundle ie. `0.0.2`.
* IMAGE_REGISTRY is the address of the registry where the build images are to be pushed to ie. `my.private.registry.com`.
* TLS_VERIFY defines whether connection to registry need TLS verification, default is `false`.
* TARGET_PLATFORM specific platform for which operator will be build. Supported values are `OCP` and `K8S`. If operator is built for other platform than `OCP`, 
then user has to manually install sriov-network-operator as described [on sriov-network-operator page](https://github.com/k8snetworkplumbingwg/sriov-network-operator). Default is `OCP`

```shell
# cd intel-ethernet-operator
# make VERSION=$(VERSION) IMAGE_REGISTRY=$(IMAGE_REGISTRY) TLS_VERIFY=$(TLS_VERIFY) TARGET_PLATFORM=K8S build_all push_all catalog-build catalog-push
```

### Installing using Operator Bundle

Once the operator images are built and accessible inside the cluster, the operator is to be installed by running the following:

Create a namespace for the operator:

```shell
# kubectl create ns intel-ethernet-operator
```

Create the following `Catalog Source` `yaml` file:

> Note: The REGISTRY_ADDRESS and VERSION need to be replaced:
>
> * VERSION is the version to be applied to the bundle ie. `0.0.1`.
> * IMAGE_REGISTRY is the address of the registry where the build images are to be pushed to ie. `my.private.registry.com`.

```yaml
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: intel-ethernet-operators
  namespace: olm
spec:
  sourceType: grpc
  image: <IMAGE_REGISTRY>/intel-ethernet-operator-catalog:<VERSION>
  publisher: Intel
  displayName: Intel ethernet operators(Local)
```

Create the `Catalog Source`

```shell
# kubectl apply -f <filename>
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
  sourceNamespace: olm
  installPlanApproval: Automatic
```

Subscribe to and install the operator:

```shell
# kubectl apply -f <filename>
```

Check that the operator is deployed:
> Note: SRIOV Network Operator pods deployed as a dependency in OCP environments.

``` 
#  kubectl -n intel-ethernet-operator get all


NAME                                                              READY   STATUS    RESTARTS   AGE
pod/clv-discovery-4vk8l                                           1/1     Running   0          22h
pod/fwddp-daemon-sjzlz                                            1/1     Running   0          22h
pod/intel-ethernet-operator-controller-manager-59645597f6-gktpm   1/1     Running   0          22h
pod/intel-ethernet-operator-controller-manager-59645597f6-jfsn9   1/1     Running   0          22h

NAME                                                         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
service/intel-ethernet-operator-controller-manager-service   ClusterIP   10.104.6.72     <none>        443/TCP   22h
service/intel-ethernet-operator-webhook-service              ClusterIP   10.98.197.202   <none>        443/TCP   22h

NAME                           DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR                                AGE
daemonset.apps/clv-discovery   1         1         1       1            1           <none>                                       22h
daemonset.apps/fwddp-daemon    1         1         1       1            1           ethernet.intel.com/intel-ethernet-present=   22h

NAME                                                         READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/intel-ethernet-operator-controller-manager   2/2     2            2           22h

NAME                                                                    DESIRED   CURRENT   READY   AGE
replicaset.apps/intel-ethernet-operator-controller-manager-59645597f6   2         2         2       22h

```

