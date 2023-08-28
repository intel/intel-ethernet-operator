```text
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2020-2023 Intel Corporation
```

# Install OOT (out of tree) ICE driver on OCP nodes

## Prerequisites

> Note: This guide was prepared and tested on environment using DCI to set up OCP.

* OCP cluster 4.12.21
* DCI configured to interact with above OCP cluster, podman installed (see note above).
* DCI agent is a privileged user (see note above).
* Redhat account with right subscription for Redhat registry access
* Internal OCP image registry is setup and configured and exposed for external access
* External image registry and its access credentials

## SSH into cluster (or OCP Controller node)

```shell
$ ssh -i <path_to_key>  dci-openshift-agent@<ip>
# or
$ oc debug node/<node_name>
```

## Install [Kernel Module Management Operator](https://openshift-kmm.netlify.app/documentation/install/)

Install KMM either from OperatorHub or from CLI using following command.

```shell
$ vi kmm.yml
```

```yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: openshift-kmm
---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: kernel-module-management
  namespace: openshift-kmm
---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: kernel-module-management
  namespace: openshift-kmm
spec:
  channel: release-1.0
  installPlanApproval: Automatic
  name: kernel-module-management
  source: redhat-operators
  sourceNamespace: openshift-marketplace
  startingCSV: kernel-module-management.v1.0.0
```

```shell
$ oc apply -f kmm.yml
```

Or just run the following to deploy the bleeding edge version:

```shell
$ oc apply -k https://github.com/rh-ecosystem-edge/kernel-module-management/config/default
```

**Warning**, OpenShift versions below 4.12 might require additional steps, see [this documentation](https://openshift-kmm.netlify.app/documentation/install/#openshift-versions-below-412)

### Verify that KMM is running in the cluster

```shell
$ oc get pods -n openshift-kmm
NAME                                                   READY   STATUS    RESTARTS   AGE
kmm-operator-controller-manager-6cff95565b-tnqwl       2/2     Running   0          10m
```

### Get Redhat image pull secret from Redhat subscription

Go to [Pull secret](https://console.redhat.com/openshift/install/pull-secret) page on Redhat OpenShift cluster manager site and download the pull secrete file. Save it on on file in accessible on client machine. Assumed it is stored in `./rht_auth.json` file. You will need to log in with your RH account.
Copy the secret to clipboard or save to a file.
Either way create the secret file on dci-agent.

```shell
$ vi ./rht_auth.json #copied from clipboard or file
```

Find out the right driver toolkit image needed for the cluster:
> Note: It is important to provide right cluster info - in case too old version is provided the latest kernel headers may not be located in the toolkit image.

```shell
$ oc adm release info 4.12.21 --image-for=driver-toolkit
quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:<some-version>
```

Pull this image locally on client machine using Podman and the authfile `./rht_auth.json` downloaded in previous step and export as variable.
`podman pull --authfile=<path to secret>  <output from above release info for driver toolkit>`

```shell
$ podman pull --authfile=./rht_auth.json quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:<some-version>
$ export OPENSHIFT_SECRET_FILE=./rht_auth.json
```

### Prepare Internal registry

#### Configure registry (Optional)

Configuring the registry as per <https://docs.openshift.com/container-platform/4.12/registry/configuring_registry_storage/configuring-registry-storage-baremetal.html>

```shell
$ oc patch configs.imageregistry.operator.openshift.io cluster --type merge --patch '{"spec":{"managementState":"Managed"}}'
```

> Note: "emptyDir" type of storage is ephemeral - in an event of a node reboot all image cache will be lost. [See following guide for more info](https://docs.openshift.com/container-platform/4.9/registry/configuring_registry_storage/configuring-registry-storage-baremetal.html)

```shell
$ oc patch configs.imageregistry.operator.openshift.io cluster --type merge --patch '{"spec":{"storage":{"emptyDir":{}}}}'
```

```shell
$ oc get pods -n openshift-image-registry                                                                  NAME                                              READY   STATUS      RESTARTS   AGE
cluster-image-registry-operator-bb55889d7-hrck2   1/1     Running     0          3d3h
image-pruner-27319680--1-tgfvf                    0/1     Completed   0          2d16h
image-pruner-27321120--1-fs9vt                    0/1     Completed   0          40h
image-pruner-27322560--1-cz827                    0/1     Completed   0          16h
image-registry-84849ff4cb-76gpb                   1/1     Running     0          40s <--------- This one
node-ca-4lb6f                                     1/1     Running     12         21d
node-ca-9bcvk                                     1/1     Running     1          21d
node-ca-cw4gf                                     1/1     Running     1          21d
node-ca-fxxhb                                     1/1     Running     1          21d
node-ca-gs4hr                                     1/1     Running     5          21d
node-ca-tnh99                                     1/1     Running     1          21d
```

#### Expose registry externally (Optional)

Exposing the registry: <https://docs.openshift.com/container-platform/4.12/registry/securing-exposing-registry.html>

```shell
$ oc patch configs.imageregistry.operator.openshift.io/cluster --patch '{"spec":{"defaultRoute":true}}' --type=merge
```

```shell
$ export HOST=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')
```

```shell
$ oc get secret -n openshift-ingress  router-certs-default -o go-template='{{index .data "tls.crt"}}' | base64 -d | sudo tee /etc/pki/ca-trust/source/anchors/${HOST}.crt  > /dev/null
```

```shell
$ sudo update-ca-trust enable
```

### Prepare driver source code image

Create new project

```shell
$ oc adm new-project oot-driver
```

```shell
$ export INTERNAL_REGISTRY=default-route-openshift-image-registry.apps.dciokd.metalkube.org
$ export EXTERNAL_REGISTRY=<MY.EXTERNAL.REGISTRY.URL>/<PEROJECT>/<REPO>
$ oc login -u admin
$ podman login -u kubeadmin -p $(oc whoami -t) $INTERNAL_REGISTRY
$ podman login -u <YOUR USER NAME> $EXTERNAL_REGISTRY 
```

Once successfully logged in to the external registry the credentials will be stored by Podman in `$XDG_RUNTIME_DIR/containers/auth.json` file (e.g. if you are are logged in as root on client machine this will be in /run/user/0/containers/auth.json)

```shell
$ export AUTH_FILE=/run/user/1000/containers/auth.json
```

Create your ICE OOT Dockerfile, provide the target kernel version, ICE version and possibly replace the URL.
The first base image should be the driver toolkit you got by running `oc adm release info 4.12.21 --image-for=driver-toolkit`.

```dockerfile
FROM <driver-toolkit> as builder

ARG KERNEL_VERSION=4.18.0-372.58.1.el8_6.x86_64
ARG ICE_VERSION=1.11.14
ENV http_proxy http://proxy-dmz.intel.com:911
ENV https_proxy http://proxy-dmz.intel.com:911
WORKDIR /usr/src
RUN ["wget", "https://downloadmirror.intel.com/772530/ice-${ICE_VERSION}.tar.gz"]
RUN ["tar","-xvf", "ice-${ICE_VERSION}.tar.gz"]
WORKDIR /usr/src/ice-${ICE_VERSION}/src
RUN ["make", "install"]

FROM registry.redhat.io/ubi9/ubi-minimal

ARG KERNEL_VERSION=4.18.0-372.58.1.el8_6.x86_64
RUN mkdir -p /opt/lib/modules/${KERNEL_VERSION}/
COPY --from=builder /usr/lib/modules/${KERNEL_VERSION}/updates/drivers/net/ethernet/intel/ice/ice.ko /opt/lib/modules/${KERNEL_VERSION}/
RUN ls  /opt/lib/modules/${KERNEL_VERSION}
RUN depmod -b /opt ${KERNEL_VERSION}
```

Build and push source container to internal registry:

```shell
$ podman build -t INTERNAL_REGISTRY/PROJECT/kmm-ice-driver:<kernel-version> .
$ podman push INTERNAL_REGISTRY/PROJECT/kmm-ice-driver:<kernel-version> 
```

### Create KMM CR

First, we need to create pull secret for external registry in oot-driver namespace so that KMMO can push images in there.

```shell
$ oc -n oot-driver create secret generic external-registry --from-file=.dockerconfigjson=/run/user/1000/containers/auth.json --type=kubernetes.io/dockerconfigjson
```

Copy and edit the below CR resource before applying it, you can find possible values with annotations [here](https://openshift-kmm.netlify.app/documentation/deploy_kmod/#example-resource).

```shell
$ vim kmm-module.yaml
```

`selector` is the label for nodes you want the driver deployed on
`regexp` is the regex which should match the kernel versions of nodes you want the driver deployed on
`containerImage` is the image name as it appears in the internal registry
`moduleName` is the name of your kernel module, it has to be ice for this module

```yaml
---
apiVersion: kmm.sigs.x-k8s.io/v1beta1
kind: Module
metadata:
  name: ice
  namespace: openshift-kmm
spec:
  moduleLoader:
    container:
      modprobe:
        moduleName: ice
      kernelMappings:
        - regexp: "4.18.0-372.58.1.el8_6.x86_64"
          containerImage: "image-registry.openshift-image-registry.svc/oot-driver/kmm-ice-driver:4.18.0-372.58.1.el8_6.x86_64"
  selector:
    node-role.kubernetes.io/worker: ""
```

Create the special resource

```shell
$ oc create -f kmm-module.yaml
```

Once the above KMMO CR is created it will start BuildConfig.

```shell
$ oc get -n openshift-kmm pod
NAME                    READY   STATUS             RESTARTS   AGE
ice-tkrzc-rtxks         1/1     Running            0          8m
```

You can now see the KMM manager logs and deployment of a DaemonSet targeting a node in the cluster.

```shell
$ oc logs -n openshift-kmm kmm-operator-controller-manager-549d9dbc84-f2rls
manager I0713 13:48:35.939340       1 module_reconciler.go:346] kmm "msg"="creating new driver container DS" "Module"={"name":"ice","namespace":"openshift-kmm"} "controller"="Module" "controllerGroup"="kmm.sigs.x-k8s.io" "controllerKin
d"="Module" "image"={"build":null,"containerImage":"image-registry.openshift-image-registry.svc:5000/kmm-ice-driver:4.18.0-372.58.1.el8_6.x86_64","literal":"","registryTLS":null,"regexp":"4.18.0-372.58.1.el8_6.x86_64"} "kernel version"
="4.18.0-372.58.1.el8_6.x86_64" "name"="ice" "namespace"="openshift-kmm" "reconcileID"="c0f582bf-e2b7-4acd-9c17-10ee798f313d"
manager I0713 13:48:35.957326       1 warning_handler.go:65] kmm/KubeAPIWarningLogger "msg"="would violate PodSecurity \"restricted:latest\": seLinuxOptions (container \"module-loader\" set forbidden securityContext.seLinuxOptions: typ
e \"spc_t\"), unrestricted capabilities (container \"module-loader\" must set securityContext.capabilities.drop=[\"ALL\"]; container \"module-loader\" must not include \"SYS_MODULE\" in securityContext.capabilities.add), restricted vol
ume types (volume \"node-lib-modules\" uses restricted volume type \"hostPath\"), runAsNonRoot != true (pod or container \"module-loader\" must set securityContext.runAsNonRoot=true), runAsUser=0 (container \"module-loader\" must not s
et runAsUser=0), seccompProfile (pod or container \"module-loader\" must set securityContext.seccompProfile.type to \"RuntimeDefault\" or \"Localhost\")"
manager I0713 13:48:35.957405       1 module_reconciler.go:358] kmm "msg"="Reconciled Driver Container" "Module"={"name":"ice","namespace":"openshift-kmm"} "controller"="Module" "controllerGroup"="kmm.sigs.x-k8s.io" "controllerKind"="M
odule" "name"="ice-tkrzc" "namespace"="openshift-kmm" "reconcileID"="c0f582bf-e2b7-4acd-9c17-10ee798f313d" "result"="created"
manager I0713 13:48:35.957439       1 module_reconciler.go:174] kmm "msg"="Handle device plugin" "Module"={"name":"ice","namespace":"openshift-kmm"} "controller"="Module" "controllerGroup"="kmm.sigs.x-k8s.io" "controllerKind"="Module"
"name"="ice" "namespace"="openshift-kmm" "reconcileID"="c0f582bf-e2b7-4acd-9c17-10ee798f313d"
manager I0713 13:48:35.957457       1 module_reconciler.go:180] kmm "msg"="Run garbage collection" "Module"={"name":"ice","namespace":"openshift-kmm"} "controller"="Module" "controllerGroup"="kmm.sigs.x-k8s.io" "controllerKind"="Module
" "name"="ice" "namespace"="openshift-kmm" "reconcileID"="c0f582bf-e2b7-4acd-9c17-10ee798f313d"
manager I0713 13:48:35.957477       1 module_reconciler.go:407] kmm "msg"="Garbage-collected DaemonSets" "Module"={"name":"ice","namespace":"openshift-kmm"} "controller"="Module" "controllerGroup"="kmm.sigs.x-k8s.io" "controllerKind"="
Module" "name"="ice" "names"=[] "namespace"="openshift-kmm" "reconcileID"="c0f582bf-e2b7-4acd-9c17-10ee798f313d"
manager I0713 13:48:35.957494       1 module_reconciler.go:415] kmm "msg"="Garbage-collected Build objects" "Module"={"name":"ice","namespace":"openshift-kmm"} "controller"="Module" "controllerGroup"="kmm.sigs.x-k8s.io" "controllerKind
"="Module" "name"="ice" "names"=null "namespace"="openshift-kmm" "reconcileID"="c0f582bf-e2b7-4acd-9c17-10ee798f313d"
manager I0713 13:48:35.957532       1 module_reconciler.go:423] kmm "msg"="Garbage-collected Sign objects" "Module"={"name":"ice","namespace":"openshift-kmm"} "controller"="Module" "controllerGroup"="kmm.sigs.x-k8s.io" "controllerKind"
="Module" "name"="ice" "names"=[] "namespace"="openshift-kmm" "reconcileID"="c0f582bf-e2b7-4acd-9c17-10ee798f313d"
manager I0713 13:48:35.962995       1 module_reconciler.go:191] kmm "msg"="Reconcile loop finished successfully" "Module"={"name":"ice","namespace":"openshift-kmm"} "controller"="Module" "controllerGroup"="kmm.sigs.x-k8s.io" "controlle
rKind"="Module" "name"="ice" "namespace"="openshift-kmm" "reconcileID"="c0f582bf-e2b7-4acd-9c17-10ee798f313d"
```
