```text
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2020-2023 Intel Corporation
```

# Intel Ethernet Operator Integration tests

**Test suite environment requirements**:

- Kubernetes Cluster:
  - 1 physical machine control node
  - 1 physical machine worker node
  - 2 CVL NICs
  - Back to back connection between 2 nodes

**Test suite environment setup**:

Please follow the steps included in the `deployment/` to set up the test environment.

**Other assumptions:**

## **CVL Discovery**

## Test 1: CVL devices are discovered correctly

Ensure that CVL devices are discovered correctly and nodes are labeled with correct values.

### Test steps

1. On each node with CVL device installed, execute

   ```shell
   $ lspci -nn | grep E810
   ```

   to get the details. Expected output:

   ```text
   18:00.0 Ethernet controller [0200]: Intel Corporation Ethernet Controller E810-C for QSFP [8086:1592] (rev 02)
   18:00.1 Ethernet controller [0200]: Intel Corporation Ethernet Controller E810-C for QSFP [8086:1592] (rev 02)
   ```

2. Supported devices are defined in `assets/100-labeler.yaml` - if node does have supported device installed, node should be labeled with

   ```shell
   $ kubectl describe node
   Name:               worker-0
   Roles:              control-plane,worker
   Labels:             ...
                       ethernet.intel.com/intel-ethernet-present=
                       ...
   ```

## **Firmware**

### Preconditions

1. Follow the [official docs](intel-ethernet-operator.md#webserver-for-disconnected-environment) to set up the webserver with NVMUPDATE package.

2. Execute

   ```shell
   $ kubectl get enc <nodename> -n intel-ethernet-operator -oyaml
   ```

   to get the information about the installed devices and their current firmware. Example output:

   ```yaml
   apiVersion: ethernet.intel.com/v1
   kind: EthernetNodeConfig
   ...
   status:
   conditions:
   - lastTransitionTime: "2022-11-29T09:00:38Z"
      message: Inventory up to date
      observedGeneration: 1
      reason: NotRequested
      status: "True"
      type: Updated
   devices:
   - DDP:
         packageName: ICE OS Default Package
         trackId: "0xc0000001"
         version: 1.3.30.0
      PCIAddress: "0000:18:00.0"
      deviceID: "1592"
      driver: ice
      driverVersion: 1.9.11
      firmware:
         MAC: b4:96:91:aa:d6:e0
         version: 3.20 0x8000d83e 1.3146.0
      name: Ethernet Controller E810-C for QSFP
      vendorID: "8086"
   ...
   ```

   Check the Firmware version on all devices on your cluster.

## Test 1: FW Update

Ensure the operator is able to update the firmware of CVL device

### Test steps

1. Follow the steps from [official docs](intel-ethernet-operator.md#updating-firmware) to create the CR and update the firmware
2. Verify if Firmware was successfully updated on the desired device on node. According to the note from docs:

   ```text
   Note: The Physical Functions of the NICs will be updated in logical pairs. The user needs to provide the FW URL and checksum (SHA-1).
   ```

   If there are more than one CVL devices installed on the node, check if only the desired one was updated.

3. Verify it Firmware has not been changed on devices on other nodes.

## Test 2: Invalid device

Verify if the operator will not conduct any operations if non-existing device is used in the CR

### Test steps

1. Create the `EthernetClusterConfig` where `deviceSelector` points to the non-existing `pciAddress`
2. Apply the CR
3. Check the status of `EthernetNodeConfig`

   ```shell
   $ kubectl get enc <nodename> -n intel-ethernet-operator -oyaml | grep reason
      reason: NotRequested
   ```

## Test 3: Invalid NVMUPDATE package

Verify if the operator will return errors when invalid input is provided

### Test steps

1. Create the `EthernetClusterConfig` where `fwURL` is set to some invalid/unreachable path and leave `fwChecksum` empty
2. Apply the CR
3. Check the status of `EthernetNodeConfig`. It should contain the information about the failure

   ```text
   status:
   conditions:
   - lastTransitionTime: "2022-12-06T10:09:34Z"
      message: 'unable to download image from: http://ice-cache.default.svc.cluster.local/invalid.tar.gzz err: 404 Not Found'
      observedGeneration: 6
      reason: Failed
      status: "False"
   ```

4. Remove the CR, modify the `fwURL` to valid one but set `fwChecksum` to some artifictial value that is not a checksum
5. Re-apply the CR
6. Verify if API returned the:

   ```text
   The EthernetClusterConfig "config" is invalid: spec.deviceConfig.fwChecksum: Invalid value: "FFFFFFFF": spec.deviceConfig.fwChecksum in body should match '^[a-fA-F0-9]{40}$'
   ```

   and the `EthernetClusterConfig` was not created:

   ```text
   kubectl get EthernetClusterConfig -n intel-ethernet-operator
   No resources found in intel-ethernet-operator namespace.
   ```

7. Remove the CR, modify the `fwURL` to valid one but set `fwChecksum` to some artifictial value
8. Re-apply the CR
9. Check the status of `EthernetNodeConfig`. It should contain the information about the failure

   ```text
   status:
   conditions:
   - lastTransitionTime: "2022-12-06T10:09:34Z"
      message: 'Checksum mismatch in downloaded file: http://ice-cache.default.svc.cluster.local/E810_NVMUpdatePackage_v4_00_Linux.tar.gz'
      observedGeneration: 8
      reason: Failed
      status: "False"
   ```

## **DDP**

### Preconditions

1. Follow the [official docs](intel-ethernet-operator.md#webserver-for-disconnected-environment) to set up the webserver with [DDP package](https://www.intel.com/content/www/us/en/download/19660/727568/intel-ethernet-800-series-dynamic-device-personalization-ddp-for-telecommunication-comms-package.html?).

2. Execute

   ```shell
   $ kubectl get enc <nodename> -n intel-ethernet-operator -oyaml
   ```

   to get the information about the installed devices and their current DDP. For example:

   ```text
     - DDP:
      packageName: ICE OS Default Package
      trackId: "0xc0000001"
      version: 1.3.30.0
   ```

## Test 1: DDP Update

Ensure the operator is able to update the DDP of CVL device

### Test steps

1. Follow the steps from [official docs](intel-ethernet-operator.md##updating-ddp) to create the CR and update the DDP
2. Verify if DDP was successfully updated on the desired device on node. According to the note from docs:

   ```text
   Note: DDP update will reboot the node
   ```

   If there are more than one CVL devices installed on the node, check if only the desired one was updated.

3. Verify it DDP has not been changed on devices on other nodes.

## Test 2: Invalid device

Verify if the operator will not conduct any operations if non-existing device is used in the CR

### Test steps

1. Create the `EthernetClusterConfig` where `deviceSelector` points to the non-existing `pciAddress`
2. Apply the CR
3. Check the status of `EthernetNodeConfig`

   ```shell
   $ kubectl get enc <nodename> -n intel-ethernet-operator -oyaml | grep reason
      reason: NotRequested
   ```

## Test 3: Invalid DDP package

Verify if the operator will return errors when invalid input is provided

### Test steps

1. Create the `EthernetClusterConfig` where `ddpURL` is set to some invalid/unreachable path and leave `ddpChecksum` empty
2. Apply the CR
3. Check the status of `EthernetNodeConfig`. It should contain the information about the failure

   ```text
   status:
   conditions:
   - lastTransitionTime: "2022-12-06T14:02:16Z"
      message: 'gzip: invalid header'
      observedGeneration: 10
      reason: Failed
      status: "False"
      type: Updated
   ```

4. Remove the CR, modify the `ddpURL` to valid one but set `ddpChecksum` to some artifictial value that is not a checksum
5. Re-apply the CR
6. Verify if API returned the:

   ```text
   The EthernetClusterConfig "ddptest" is invalid: spec.deviceConfig.ddpChecksum: Invalid value: "FFFFFF": spec.deviceConfig.ddpChecksum in body should match '^[a-fA-F0-9]{40}$'
   ```

   and the `EthernetClusterConfig` was not created:

   ```shell
   $ kubectl get EthernetClusterConfig -n intel-ethernet-operator
   No resources found in intel-ethernet-operator namespace.
   ```

7. Remove the CR, modify the `ddpURL` to valid one but set `ddpChecksum` to some artifictial value
8. Re-apply the CR
9. Check the status of `EthernetNodeConfig`. It should contain the information about the failure

   ```text
   status:
   conditions:
   - lastTransitionTime: "2022-12-06T10:09:34Z"
      message: 'Checksum mismatch in downloaded file: http://ice-cache.default.svc.cluster.local/ice_comms-1.3.35.0.zip'
      observedGeneration: 8
      reason: Failed
      status: "False"
   ```
