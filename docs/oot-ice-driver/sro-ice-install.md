```text
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2021 Intel Corporation
```

# Install OOT (out of tree) ICE driver on OCP nodes

## Prerequisites

> Note: This guide was prepared and tested on environment using DCI to set up OCP.

* OCP cluster 4.9.7
* DCI configured to interact with above OCP cluster, podman installed (see note above).
* DCI agent is a privileged user (see note above).
* Redhat account with right subscription for Redhat registry access
* Internal OCP image registry is setup and configured and exposed for external access
* External image registry and its access credentials
* NFD operator is deployed and NFD deployment must be created either using `oc cli` or `console dashboard`. (Operator gets deployed with the SRO, NFD deployment must be deployed manually)

## SSH into cluster (or OCP Controller node)

```shell
 ssh -i <path_to_key>  dci-openshift-agent@<ip>
```

## Install [Special Resource Operator](https://github.com/openshift/special-resource-operator)

Install SRO either from OperatorHub or from CLI using following command.

```shell
# vi sro.yml
```

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: openshift-special-resource-operator

---

apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: special-resource-operators
  namespace: openshift-special-resource-operator
spec:
  targetNamespaces:
    - openshift-special-resource-operator

---

apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: sro-operator-subscription
  namespace: openshift-special-resource-operator
spec:
  channel: "4.9"
  name: openshift-special-resource-operator
  source: "redhat-operators"
  sourceNamespace: openshift-marketplace
```

```shell
# oc apply -f sro.yml
```

### Install NFD

```shell
# vi nfd-cr.yml
```

```yaml
apiVersion: nfd.openshift.io/v1
kind: NodeFeatureDiscovery
metadata:
  name: nfd-instance
  namespace: openshift-special-resource-operator
spec:
  instance: "" # instance is empty by default
  operand:
    namespace: openshift-special-resource-operator
    image: quay.io/openshift/origin-node-feature-discovery:4.8
    imagePullPolicy: Always
  workerConfig:
    configData: |
      #core:
      #  labelWhiteList:
      #  noPublish: false
      #  sleepInterval: 60s
      #  sources: [all]
      #  klog:
      #    addDirHeader: false
      #    alsologtostderr: false
      #    logBacktraceAt:
      #    logtostderr: true
      #    skipHeaders: false
      #    stderrthreshold: 2
      #    v: 0
      #    vmodule:
      ##   NOTE: the following options are not dynamically run-time configurable
      ##         and require a nfd-worker restart to take effect after being changed
      #    logDir:
      #    logFile:
      #    logFileMaxSize: 1800
      #    skipLogHeaders: false
      #sources:
      #  cpu:
      #    cpuid:
      ##     NOTE: whitelist has priority over blacklist
      #      attributeBlacklist:
      #        - "BMI1"
      #        - "BMI2"
      #        - "CLMUL"
      #        - "CMOV"
      #        - "CX16"
      #        - "ERMS"
      #        - "F16C"
      #        - "HTT"
      #        - "LZCNT"
      #        - "MMX"
      #        - "MMXEXT"
      #        - "NX"
      #        - "POPCNT"
      #        - "RDRAND"
      #        - "RDSEED"
      #        - "RDTSCP"
      #        - "SGX"
      #        - "SSE"
      #        - "SSE2"
      #        - "SSE3"
      #        - "SSE4.1"
      #        - "SSE4.2"
      #        - "SSSE3"
      #      attributeWhitelist:
      #  kernel:
      #    kconfigFile: "/path/to/kconfig"
      #    configOpts:
      #      - "NO_HZ"
      #      - "X86"
      #      - "DMI"
      #  pci:
      #    deviceClassWhitelist:
      #      - "0200"
      #      - "03"
      #      - "12"
      #    deviceLabelFields:
      #      - "class"
      #      - "vendor"
      #      - "device"
      #      - "subsystem_vendor"
      #      - "subsystem_device"
      #  usb:
      #    deviceClassWhitelist:
      #      - "0e"
      #      - "ef"
      #      - "fe"
      #      - "ff"
      #    deviceLabelFields:
      #      - "class"
      #      - "vendor"
      #      - "device"
      #  custom:
      #    - name: "my.kernel.feature"
      #      matchOn:
      #        - loadedKMod: ["example_kmod1", "example_kmod2"]
      #    - name: "my.pci.feature"
      #      matchOn:
      #        - pciId:
      #            class: ["0200"]
      #            vendor: ["15b3"]
      #            device: ["1014", "1017"]
      #        - pciId :
      #            vendor: ["8086"]
      #            device: ["1000", "1100"]
      #    - name: "my.usb.feature"
      #      matchOn:
      #        - usbId:
      #          class: ["ff"]
      #          vendor: ["03e7"]
      #          device: ["2485"]
      #        - usbId:
      #          class: ["fe"]
      #          vendor: ["1a6e"]
      #          device: ["089a"]
      #    - name: "my.combined.feature"
      #      matchOn:
      #        - pciId:
      #            vendor: ["15b3"]
      #            device: ["1014", "1017"]
      #          loadedKMod : ["vendor_kmod1", "vendor_kmod2"]
  customConfig:
    configData: |
      #    - name: "more.kernel.features"
      #      matchOn:
      #      - loadedKMod: ["example_kmod3"]
      #    - name: "more.features.by.nodename"
      #      value: customValue
      #      matchOn:
      #      - nodename: ["special-.*-node-.*"]

```

```shell
# oc apply -f nfd-cr.yml
```

### Verify that SRO and NFD is running in the cluster

```shell
# oc get pods -n openshift-special-resource-operator
NAME                                                   READY   STATUS    RESTARTS   AGE
nfd-controller-manager-5d6f477c44-kdk5d                2/2     Running   0          2d21h
nfd-master-7qkm8                                       1/1     Running   0          5m28s
nfd-master-c629s                                       1/1     Running   0          5m28s
nfd-master-rq55j                                       1/1     Running   0          5m28s
nfd-worker-4v747                                       1/1     Running   0          5m28s
nfd-worker-k85cc                                       1/1     Running   0          5m28s
nfd-worker-s59nj                                       1/1     Running   0          5m28s
special-resource-controller-manager-5557cdcf55-wpbqs   2/2     Running   0          2d21h
```

> Note: If SRO is installed via OCP console dashboard, the NFD dependency also will be installed on same namespace openshift-special-resource-operator.

### Get Redhat image pull secret from Redhat subscription

Go to [Pull secret](https://console.redhat.com/openshift/install/pull-secret) page on Redhat OpenShift cluster manager site and download the pull secrete file. Save it on on file in accessible on client machine. Assumed it is stored in `./rht_auth.json` file. You will need to log in with your RH account.
Copy the secret to clipboard or save to a file.
Either way create the secret file on dci-agent.

```shell
vi ./rht_auth.json #copied from clipboard or file
```

Find out the right driver toolkit image needed for the cluster:
> Note: It is important to provide right cluster info - in case too old version is provided the latest kernel headers may not be located in the toolkit image.

```shell
# oc adm release info 4.9.7 --image-for=driver-toolkit
quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:65369b6ccff0f3dcdaa8685ce4d765b8eba0438a506dad5d4221770a6ac1960a
```

Pull this image locally on client machine using Podman and the authfile `./rht_auth.json` downloaded in previous step and export as variable.
`podman pull --authfile=<path to secret>  <output from above release info for driver toolkit>`

```shell
# podman pull --authfile=./rht_auth.json quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:65369b6ccff0f3dcdaa8685ce4d765b8eba0438a506dad5d4221770a6ac1960a
# export OPENSHIFT_SECRET_FILE=./rht_auth.json
```

### Prepare Internal registry

#### Configure registry (Optional)

Configuring the registry as per https://docs.openshift.com/container-platform/4.9/registry/configuring_registry_storage/configuring-registry-storage-baremetal.html

```shell
# oc patch configs.imageregistry.operator.openshift.io cluster --type merge --patch '{"spec":{"managementState":"Managed"}}'
```

> Note: "emptyDir" type of storage is ephemeral - in an event of a node reboot all image cache will be lost. [See following guide for more info](https://docs.openshift.com/container-platform/4.9/registry/configuring_registry_storage/configuring-registry-storage-baremetal.html)

```shell
# oc patch configs.imageregistry.operator.openshift.io cluster --type merge --patch '{"spec":{"storage":{"emptyDir":{}}}}'
```

```shell
oc get pods -n openshift-image-registry                                                                  NAME                                              READY   STATUS      RESTARTS   AGE
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

Exposing the registry: https://docs.openshift.com/container-platform/4.9/registry/securing-exposing-registry.html

```shell
# oc patch configs.imageregistry.operator.openshift.io/cluster --patch '{"spec":{"defaultRoute":true}}' --type=merge
```

```shell
# export HOST=$(oc get route default-route -n openshift-image-registry --template='{{ .spec.host }}')
```

```shell
# oc get secret -n openshift-ingress  router-certs-default -o go-template='{{index .data "tls.crt"}}' | base64 -d | sudo tee /etc/pki/ca-trust/source/anchors/${HOST}.crt  > /dev/null
```

```shell
# sudo update-ca-trust enable
```

### Prepare driver source code image

Create new project

```shell
# oc adm new-project oot-driver
```

Clone [OpenShift CNF feature deploy](https://github.com/openshift-kni/cnf-features-deploy/) repo (tested with commit: 1167aca8c60abfb61bbe65013771890708f5d55b).

```shell
# git clone https://github.com/openshift-kni/cnf-features-deploy.git
# cd cnf-feature-deploy/tools/oot-driver/
```

Edit the ICE_DRIVER_VERSION from 1.6.4 to 1.6.7
In files:

* tools/oot-driver/charts/ice-driver-0.0.1/values.yaml
* tools/oot-driver/templates/special-resources/ice-driver-special-resource.yaml.template

```yaml
  - Name: ICE_DRIVER_VERSION
    Value: 1.6.7
```

Download ICE driver source code (tested with v1.6.7).

```shell
# curl https://netix.dl.sourceforge.net/project/e1000/ice%20stable/1.6.7/ice-1.6.7.tar.gz -o ice-1.6.7.tar.gz
# mv ice-1.6.7.tar.gz files/driver
```

Login to OCP internal registry as well as external registry using podman.
(Where dciokd.metalkube.org is the `<cluster URL>` in `default-route-openshift-image-registry.apps.<cluster URL>`, this will differ on non DCI cluster)
(External registry was already configure at time of testing)

```shell
# export INTERNAL_REGISTRY=default-route-openshift-image-registry.apps.dciokd.metalkube.org
# export EXTERNAL_REGISTRY=<MY.EXTERNAL.REGISTRY.URL>/<PEROJECT>/<REPO>
# oc login -u admin
# podman login -u kubeadmin -p $(oc whoami -t) $INTERNAL_REGISTRY
# podman login -u <YOUR USER NAME> $EXTERNAL_REGISTRY 
```

Once successfully logged in to the external registry the credentials will be stored by Podman in `$XDG_RUNTIME_DIR/containers/auth.json` file (e.g. if you are are logged in as root on client machine this will be in /run/user/0/containers/auth.json)

```shell
# export AUTH_FILE=/run/user/1000/containers/auth.json
```

Build and push source container to internal registry:

```shell
# make build-source-container && make push-source-container
```

Add sudo privilege to Makefile in helm-create-config-map, to do that edit Makefile (line 64, 65):

```shell
     56 helm:
     57 ifeq (, $(shell which helm))
     58         @{ \
     59         set -e ;\
     60         HELM_GEN_TMP_DIR=$$(mktemp -d) ;\
     61         cd $$HELM_GEN_TMP_DIR ;\
     62         curl https://get.helm.sh/helm-v3.6.0-linux-amd64.tar.gz -o helm.tar.gz ;\
     63         tar xvfpz helm.tar.gz ;\
     64         sudo mv linux-amd64/helm /usr/local/bin ;\
     65         sudo chmod +x /usr/local/bin/helm ;\
     66         rm -rf $$HELM_GEN_TMP_DIR ;\
     67         }

```

Build and upload the OOT driver helm chart:

```shell
# make helm-create-config-map
```

### Build SRO CR

First, we need to create pull secret for external registry in oot-driver namespace so that SRO can push images in there.

```shell
# oc -n oot-driver create secret generic external-registry --from-file=.dockerconfigjson=/run/user/1000/containers/auth.json --type=kubernetes.io/dockerconfigjson
```

Edit special-resource.yaml after it's created. We only want to keep oot-ice driver SRO config. Comment out or delete everything else.

```shell
# cp ./templates/special-resources/ice-driver-special-resource.yaml.template ./special-resource.yaml
# vi ./special-resource.yaml
```

Where:
`externalRegistry` is the name of the image as it will appear in external registry
`kernelVersion` is the kernel version on my worker nodes that will have driver updated
`àrtifacts.images.name` is the image name as it appears in the internal registry

```yaml
apiVersion: sro.openshift.io/v1beta1
kind: SpecialResource
metadata:
  generateName: ice-driver-
spec:
  namespace: oot-driver
  chart: 
    name: ice-driver
    version: 0.0.1
    repository:
      name: chart
      url: cm://oot-driver/charts
  set:
    kind: Values
    apiVersion: sro.openshift.io/v1beta1
    kmodNames: ["ice"]
    containerName: "ice-driver-container"
    externalRegistry: "ger-is-registry.caas.intel.com/openness-operators"
    signDriver: false
    downloadDriver: false
    kernelVersion: 4.18.0-305.25.1.rt7.97.el8_4.x86_64
    buildArgs:
    - Name: "KMODVER"
      Value: "SRO"
    - Name: "IMAGE"
      Value: quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:65369b6ccff0f3dcdaa8685ce4d765b8eba0438a506dad5d4221770a6ac1960a
    - Name: "KERNEL_SOURCE"
      Value: "yum"
    - Name: "ICE_DRIVER_VERSION"
      Value: "1.6.7"
  driverContainer:
    artifacts:
      images:
        - name: "oot-source-driver:latest"
          kind: ImageStreamTag
          namespace: "oot-driver"
          path:
            - sourcePath: "/usr/src/oot-driver/."
              destinationDir: "./"
```

#### Create ICE OOT SRO

Create the special resource
```shell
# oc create -f special-resource.yaml
```
Once the above SRO CR is created it will start BuildConfig.

```shell
# oc get bc -A
NAMESPACE    NAME                                TYPE     FROM   LATEST
oot-driver   ice-driver-9z9j4-bfb16b50984f16f0   Docker          1
oot-driver   ice-driver-6mwgt-efb0a5d31af5b3fd   Docker          1
```

You can now see the ICE driver build logs from the build config containers:

```shell
# oc logs -f bc/ice-driver-9z9j4-bfb16b50984f16f0
Adding cluster TLS certificate authority to trust store
Caching blobs under "/var/cache/blobs".
Getting image source signatures
Copying blob sha256:54d93546ee64d7e43b068c6dd2426e49c20aa031458a58c79ebc7020a791e564
Copying blob sha256:6ea6b01ba32a53862e954f377e01e3116b8f9447c95cd36f21c86d7dc73964e6
Copying blob sha256:22f677655049d4c2e6cd9e49ca9ed20f34ac175ef0c82f5c5eabc79031c1c29a
Copying config sha256:96f0fd41c617413fc87d82c015d6a808b44500a985884eb3de282a4eeb050ddf
Writing manifest to image destination
Storing signatures
Adding cluster TLS certificate authority to trust store
Adding cluster TLS certificate authority to trust store
time="2021-12-14T14:07:00Z" level=info msg="Not using native diff for overlay, this may cause degraded performance for building images: kernel has CONFIG_OVERLAY_FS_REDIRECT_DIR enabled"
I1214 14:07:00.166070       1 defaults.go:102] Defaulting to storage driver "overlay" with options [mountopt=metacopy=on].
Caching blobs under "/var/cache/blobs".
Adding transient rw bind mount for /run/secrets/rhsm
STEP 1: FROM quay.io/openshift-release-dev/ocp-v4.0-art-dev@sha256:65369b6ccff0f3dcdaa8685ce4d765b8eba0438a506dad5d4221770a6ac1960a AS builder
Getting image source signatures
Copying blob sha256:95bb11286ffa4629626361161846b587eaebf7e6fc5be9b59120388d459a1aea
Copying blob sha256:eac1b95df832dc9f172fd1f07e7cb50c1929b118a4249ddd02c6318a677b506a
Copying blob sha256:47aa3ed2034c4f27622b989b26c06087de17067268a19a1b3642a7e2686cd1a3
Copying blob sha256:450c1856cde291bbe8de8e7eff490e70c10d4bc6f3f51c1f4f6170bd529ca925
Copying blob sha256:930a24a335705c0b20fe79b020f568f998354e975373e27ce42982fd3aa323fd
Copying config sha256:1ba2642807db05e70de7ecde10dddf519b48891ef334948049b1bb742855fb1b
Writing manifest to image destination
Storing signatures
STEP 2: ARG KVER
--> 543d88b47b1
STEP 3: ENV KVER=$KVER
--> 20a8cdccf35
STEP 4: ARG KERNEL_SOURCE
--> ae1b4d70cf8
STEP 5: ENV KERNEL_SOURCE=$KERNEL_SOURCE
--> d2c86ff2abc
STEP 6: ARG ICE_DRIVER_VERSION

...

STEP 19: RUN tar zxf ice-$ICE_DRIVER_VERSION.tar.gz
--> b8377046140
STEP 20: WORKDIR ice-$ICE_DRIVER_VERSION/src
--> fbc23b5afdd
STEP 21: RUN BUILD_KERNEL=$KVER KSRC=/lib/modules/$KVER/build/ make modules_install
echo "*** The target kernel has CONFIG_MODULE_SIG_ALL enabled, but" ; echo "*** the signing key cannot be found. Module signing has been" ; echo "*** disabled for this build." ; make  ccflags-y="" -C "/lib/modules/4.18.0-305.25.1.rt7.97.el8_4.x86_64/build/" CONFIG_ICE=m CONFIG_MODULE_SIG=n CONFIG_MODULE_SIG_ALL= M="/tmp/kmods-via-containers/files/driver/ice-1.6.7/src"    modules
*** The target kernel has CONFIG_MODULE_SIG_ALL enabled, but
*** the signing key cannot be found. Module signing has been
*** disabled for this build.
make[1]: Entering directory '/usr/src/kernels/4.18.0-305.25.1.rt7.97.el8_4.x86_64'
  CC [M]  /tmp/kmods-via-containers/files/driver/ice-1.6.7/src/ice_main.o
  CC [M]  /tmp/kmods-via-containers/files/driver/ice-1.6.7/src/ice_controlq.o
  CC [M]  /tmp/kmods-via-containers/files/driver/ice-1.6.7/src/ice_common.o
  CC [M]  /tmp/kmods-via-containers/files/driver/ice-1.6.7/src/ice_nvm.o
  CC [M]  /tmp/kmods-via-containers/files/driver/ice-1.6.7/src/ice_switch.o
  CC [M]  /tmp/kmods-via-containers/files/driver/ice-1.6.7/src/ice_sched.o

...

STEP 33: CMD ["/entrypoint.sh"]
--> bdcca86925b
STEP 34: ENV "OPENSHIFT_BUILD_NAME"="ice-driver-dt58h-efb0a5d31af5b3fd-1" "OPENSHIFT_BUILD_NAMESPACE"="oot-driver"
--> 3f1aec0e379
STEP 35: LABEL "io.openshift.build.name"="ice-driver-dt58h-efb0a5d31af5b3fd-1" "io.openshift.build.namespace"="oot-driver"
STEP 36: COMMIT temp.builder.openshift.io/oot-driver/ice-driver-dt58h-efb0a5d31af5b3fd-1:020901a7
--> d38130cd7e0
[Warning] one or more build args were not consumed: [KMODVER]
d38130cd7e03083560f65b95eb873106ae5de3e261ceb88bbca2076b22a087fe

Pushing image ger-is-registry.caas.intel.com/openness-operators/damian/ice-rt-sro/ice-driver-container:4.18.0-305.25.1.rt7.97.el8_4.x86_64 ...
Getting image source signatures
Copying blob sha256:970f84205eb3b5b84aa989dccfb68111999322fd493b90be07751baa1669a406
Copying blob sha256:d18cd35c5ef6cc34ba164906e05f099e2f339ac777f07b6bb512e8663746a23d
Copying blob sha256:a096d84da1f73ed5bb3db4e44cb0ceffe0c5e887744179aec44c291c045e0db9
Copying blob sha256:c2fa1a87414966a6c22628104e86228c85aa7d9fe12fbbffc378d242d3acee45
Copying blob sha256:0d3f22d60daf4a2421b2239fb0e1c6ec02d3787274db8b098fb648941ea2d5dc
Copying blob sha256:0488bd866f642b2b1b5490f5c50d628815e4e8fa1f7cae57d52c67c1e9d3e2cc
Copying blob sha256:82aa9533e187a5d64585b9e2fe74bf038b2def24ecbb2a5fcf95ce86f26dee77
Copying blob sha256:f7cb630a5e562f964c6c3eae244ebac6d219490d318ab2e96bf38f1030065542
Copying blob sha256:2320a5df52179bec59cb284c1a2e0f35f74a9db5447bcac1233edefc74d8a28c
Copying config sha256:d38130cd7e03083560f65b95eb873106ae5de3e261ceb88bbca2076b22a087fe
Writing manifest to image destination
Storing signatures
Successfully pushed ger-is-registry.caas.intel.com/openness-operators/damian/ice-rt-sro/ice-driver-container@sha256:f4fb622791591587c40c41b02b94321538f889a27a31f147575b52f21c65145a
Push successful
```

### Install OOT ICE module on host

Create and apply MachineConfig to install the ICE OOT driver on your node.
Important thing to look out for is the NODE\_LABEL in Makefile.

This NODE\_LABEL needs to match the MachineConfigPool the worker nodes are on.

```shell
vi Makefile

# change node label to this (replace with actual label for all nodes marked with the label)
NODE_LABEL ?= "machineconfiguration.openshift.io/role: worker-rt"
```

```shell
# make create-machine-config
```

Edit the generated `oot-driver-machine-config.yaml` file to comment out systemd unit contents for other kernel modules.

Where:
`machineconfiguration.openshift.io/role` matches the worker role.
`ExecStart=/usr/bin/bash` line 35 point to the ice driver image pushed to the kernel repository
`ExecStop=/usr/bin/bash` line 36 point to the ice driver image pushed to the kernel repository
For example:

```yaml
      1 ---
      2 apiVersion: machineconfiguration.openshift.io/v1
      3 kind: MachineConfig
      4 metadata:
      5   labels:
      6     machineconfiguration.openshift.io/role: worker-rt
      7   name: 10-oot-driver-loading
      8 spec:
      9   config:
     10     ignition: {version: 2.2.0}
     11     storage:
     12       files:
     13         - contents: {source: 'data:text/plain;charset=us-ascii;base64,IyEvYmluL2Jhc2gKc2V0IC1ldQoKQUNUSU9OPSQxOyBzaGlmdApJTUFHRT0kMTsgc2hpZnQKS0VSTkVMPWB1bmFtZSAtcmAKCnBvZG1hbiBwdWxsIC0tYXV0aGZpbGUgL3Zhci9saWIva3ViZWxldC9jb25maWcuanNvbiAke0lNQUdFfToke0t        FUk5FTH0gMj4mMQoKbG9hZF9rbW9kcygpIHsKCiAgICBwb2RtYW4gcnVuIC1pIC0tcHJpdmlsZWdlZCAtdiAvbGliL21vZHVsZXMvJHtLRVJORUx9L2tlcm5lbC9kcml2ZXJzLzovbGliL21vZHVsZXMvJHtLRVJORUx9L2tlcm5lbC9kcml2ZXJzLyAke0lNQUdFfToke0tFUk5FTH0gbG9hZC5zaAp9CnVubG9hZF9rbW9kcygpIHsKICAg        IHBvZG1hbiBydW4gLWkgLS1wcml2aWxlZ2VkIC12IC9saWIvbW9kdWxlcy8ke0tFUk5FTH0va2VybmVsL2RyaXZlcnMvOi9saWIvbW9kdWxlcy8ke0tFUk5FTH0va2VybmVsL2RyaXZlcnMvICR7SU1BR0V9OiR7S0VSTkVMfSB1bmxvYWQuc2gKfQoKY2FzZSAiJHtBQ1RJT059IiBpbgogICAgbG9hZCkKICAgICAgICBsb2FkX2ttb2RzC        iAgICA7OwogICAgdW5sb2FkKQogICAgICAgIHVubG9hZF9rbW9kcwogICAgOzsKICAgICopCiAgICAgICAgZWNobyAiVW5rbm93biBjb21tYW5kLiBFeGl0aW5nLiIKICAgICAgICBlY2hvICJVc2FnZToiCiAgICAgICAgZWNobyAiIgogICAgICAgIGVjaG8gImxvYWQgICAgICAgIExvYWQga2VybmVsIG1vZHVsZShzKSIKICAgICAgIC        BlY2hvICJ1bmxvYWQgICAgICBVbmxvYWQga2VybmVsIG1vZHVsZShzKSIKICAgICAgICBleGl0IDEKZXNhYwo='}
     14           filesystem: root
     15           mode: 493
     16           path: /usr/local/bin/oot-driver
     17     systemd:
     18       units:
     19       - contents: |
     20           [Unit]
     21           Description=out of tree ice-driver loader
     22           # Start after the network is up
     23           Wants=network-online.target
     24           After=network-online.target
     25           # Also after docker.service (no effect on systems without docker)
     26           After=docker.service
     27           # Before kubelet.service (no effect on systems without kubernetes)
     28           Before=kubelet.service
     29
     30           [Service]
     31           Type=oneshot
     32           TimeoutStartSec=25m
     33           RemainAfterExit=true
     34           # Use bash to workaround https://github.com/coreos/rpm-ostree/issues/1936
     35           ExecStart=/usr/bin/bash -c "/usr/local/bin/oot-driver load ger-is-registry.caas.intel.com/openness-operators/damian/ice-rt-sro/ice-driver-container"
     36           ExecStop=/usr/bin/bash -c "/usr/local/bin/oot-driver unload ger-is-registry.caas.intel.com/openness-operators/damian/ice-rt-sro/ice-driver-container"
     37           StandardOutput=journal+console
     38
     39           [Install]
     40           WantedBy=default.target
     41         enabled: true
     42         name: "oot-ice-driver-load.service"
```

Create the machine config:

```shell
# oc apply -f oot-driver-machine-config.yaml
```

### Verification

Login to a worker node and check with systemctl:

```shell
# ssh -i <path_to_key>  dci-openshift-agent@<ip>
# ssh dci@provisionhost
# ssh core@worker-1
```

```shell
[core@worker-1 ~]$ sudo systemctl status oot-ice-driver-load.service
● oot-ice-driver-load.service - out of tree ice-driver loader
   Loaded: loaded (/etc/systemd/system/oot-ice-driver-load.service; enabled; vendor preset: disabled)
   Active: active (exited) since Wed 2021-12-08 23:44:08 UTC; 1 day 14h ago
 Main PID: 5967 (code=exited, status=0/SUCCESS)
    Tasks: 0 (limit: 3297305)
   Memory: 322.1M
      CPU: 14.988s
   CGroup: /system.slice/oot-ice-driver-load.service

Dec 08 23:43:41 worker-1 bash[5967]: Copying blob sha256:5599d717b80921fb41ccbdeb7a997e679453c9a5611977e563e77b2ef466bfa4
Dec 08 23:43:41 worker-1 bash[5967]: Copying blob sha256:cf39a0372dd22c5bcb1d652cf8112d2965f940d991561b426c9700041195f87e
Dec 08 23:43:53 worker-1 bash[5967]: Copying config sha256:377cba53aad5e275a37678062cebe8e68f6c161b619d02830aae5860a7e73910
Dec 08 23:43:53 worker-1 bash[5967]: Writing manifest to image destination
Dec 08 23:43:53 worker-1 bash[5967]: Storing signatures
Dec 08 23:43:57 worker-1 bash[5967]: 377cba53aad5e275a37678062cebe8e68f6c161b619d02830aae5860a7e73910
Dec 08 23:43:58 worker-1 bash[5967]: depmod: WARNING: could not open /lib/modules/4.18.0-305.19.1.el8_4.x86_64/modules.order: No such file or directory
Dec 08 23:44:01 worker-1 bash[5967]: depmod: WARNING: could not open /lib/modules/4.18.0-305.19.1.el8_4.x86_64/modules.builtin: No such file or directory
Dec 08 23:44:07 worker-1 bash[5967]: oot ice driver loaded
Dec 08 23:44:08 worker-1 systemd[1]: Started out of tree ice-driver loader.

```

Check that a know CLV device is bound to ICE driver:

```shell
#ethtool -i ens5f0
driver: ice
version: 1.6.7
firmware-version: 3.00 0x80008944 20.5.13
expansion-rom-version:
bus-info: 0000:b1:00.0
supports-statistics: yes
supports-test: yes
supports-eeprom-access: yes
supports-register-dump: yes
supports-priv-flags: yes
```
