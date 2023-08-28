```text
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2020-2023 Intel Corporation
```

# Install OOT (out of tree) ICE driver on Kubernetes nodes

## Prerequisites

* Kubernetes cluster
* External image registry and its access credentials
* Docker or Podman to build images

## Install [Kernel Module Management Operator](https://kmm.sigs.k8s.io/documentation/install/)

Install cert-manager dependency (if not installed already)

```shell
$ kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.11.0/cert-manager.yaml
$ kubectl -n cert-manager wait --for=condition=Available deployment \
    cert-manager \
    cert-manager-cainjector \
    cert-manager-webhook
```

Install KMM

```shell
$ kubectl apply -k https://github.com/kubernetes-sigs/kernel-module-management/config/default
```

### Verify that KMM is running in the cluster

```shell
$ kubectl get pods -n kmm-operator-system
NAME                                                   READY   STATUS    RESTARTS   AGE
kmm-operator-controller-manager-6cff95565b-tnqwl       2/2     Running   0          10m
```

### Prepare driver source code image

KMMO requires that you build an image, called ModuleLoader image, which will have the .ko file - kernel module, in the /opt directory.
It also has to have the ``kmod`` utility installed, namely ``modprobe`` and ``sleep`` command. Create your ICE OOT Dockerfile, provide the target kernel version, ICE version and possibly replace the URL. You should also use your target OS as the base images and the equivalent dependencies.

```dockerfile
FROM ubuntu as builder
 
ARG KERNEL_VERSION=5.15.0-73-generic
ARG ICE_VERSION=1.11.14
ENV http_proxy <http_proxy>
ENV https_proxy <https_proxy>
RUN apt-get update && apt-get install -y bc \
    bison \
    flex \
    libelf-dev \
    gnupg \
    wget \
    tar \
    git \
    make \
    gcc \
    linux-generic \
    linux-headers-${KERNEL_VERSION} \
    linux-modules-${KERNEL_VERSION} \
    linux-modules-extra-${KERNEL_VERSION}
WORKDIR /usr/src
RUN ["wget", "https://downloadmirror.intel.com/772530/ice-${ICE_VERSION}.tar.gz"]
RUN ["tar","-xvf", "ice-${ICE_VERSION}.tar.gz"]
WORKDIR /usr/src/ice-${ICE_VERSION}/src
RUN ["make", "install"]
 
FROM ubuntu
 
ARG KERNEL_VERSION=5.15.0-73-generic
RUN apt-get update && apt-get install -y kmod
RUN mkdir -p /opt/lib/modules/${KERNEL_VERSION}/
 
COPY --from=builder /usr/lib/modules/${KERNEL_VERSION}/kernel/drivers/net/ethernet/intel/ice/ice.ko /opt/lib/modules/${KERNEL_VERSION}/
RUN ls  /opt/lib/modules/${KERNEL_VERSION}
RUN depmod -b /opt ${KERNEL_VERSION}
```

Build and push source container to registry:

```shell
$ podman build -t <registry>/ice-driver-kernel-module:<kernel-version> .
$ podman push <registry>/ice-driver-kernel-module:<kernel-version> 
```

### Create KMM CR

```shell
$ vim kmm-module.yaml
```

`selector` is the label for nodes you want the driver deployed on
`regexp` is the regex which should match the kernel versions of nodes you want the driver deployed on
`containerImage` is the image name as it appears in the internal registry
`moduleName` is the name of your kernel module, it has to be ice for this module.

```yaml
---
apiVersion: kmm.sigs.x-k8s.io/v1beta1
kind: Module
metadata:
  name: ice
  namespace: kmm-operator-system
spec:
  moduleLoader:
    container:
      modprobe:
        moduleName: ice
        inTreeModuleToRemove: ice
      kernelMappings:
        - regexp: '5.15.0-73-generic'
          containerImage: <registry>/ice-driver-kernel-module:<kernel-version>
  selector:
    node-role.kubernetes.io/worker: ""
```

> Note: To load different version of ice module, first old version of it needs to be unloaded. KMMO supports unloading of old module, but when the ModuleLoader pod is terminated, there is a limitation that the old module won't be loaded again.

Create the special resource

```shell
$ kubectl create -f kmm-module.yaml
```

Once the above KMMO CR is created it will start BuildConfig.

```shell
$ kubectl get -n kmm-operator-system pod
NAME                    READY   STATUS             RESTARTS   AGE
ice-tkrzc-rtxks         1/1     Running            0          8m
```

You can now see the KMM manager logs and deployment of a DaemonSet targeting a node in the cluster.
