```text
SPDX-License-Identifier: Apache-2.0
Copyright (c) 2020-2021 Intel Corporation
```

# Creating NodeFlowConfig Spec
To apply flow rules, a resource of type NodeFlowConfig needs to be created. At the moment the Unified Flow Operator gives only partial support of the Generic flow API. All the supported options are described below.

NOTE: Most of the objects parameters names are consistent with the names given in the [official dpdk rte flow documentation](https://doc.dpdk.org/guides/prog_guide/rte_flow.html). 

For the full description of Generic flow API see https://doc.dpdk.org/guides/prog_guide/rte_flow.html.

## Example NodeFlowConfig
A correct NodeFlowConfig should be similar to this:

```yaml
apiVersion: flowconfig.intel.com/v1
kind: NodeFlowConfig
metadata:
  name: node1
spec:
  rules:
    - pattern:
        - type: RTE_FLOW_ITEM_TYPE_ETH
          spec:
            dst: 00:00:12:23:00:02
          last:
            dst: 00:00:12:23:00:10
        - type: RTE_FLOW_ITEM_TYPE_UDP
          spec:
            hdr:
              dst: 67
        - type: RTE_FLOW_ITEM_TYPE_END
      action:
        - type: RTE_FLOW_ACTION_TYPE_VF
          conf:
            id: 1
        - type: RTE_FLOW_ACTION_TYPE_END
      attr:
        priority: 0
        ingress: 1
```

NOTE: Make sure to use the correct names of the types and their parameters.

## Flow Rules
A flow rule is a set of attributes, matching pattern and a list of actions. Port Id is the port identifier of the used Ethernet device.
- PortId
-	Attributes
-	Pattern
-	Action

### Pattern Item
Pattern item can match a specific packet data or traffic properties. It can also describe properties of the pattern.

An Item can contain up to three structures of the same type:
- spec
- last
- mask

#### Supported Item Types
At the moment Unified Flow Operator supports item types listed below.

|   Item                                |   Description                         |
|---------------------------------------|---------------------------------------|
|   RTE_FLOW_ITEM_TYPE_ETH              |   Ethernet header                     |
|   RTE_FLOW_ITEM_TYPE_VLAN             |   802.1Q/ad VLAN tag                  |
|   RTE_FLOW_ITEM_TYPE_IPV4             |   IPv4 header                         |
|   RTE_FLOW_ITEM_TYPE_UDP              |   UDP header                          |
|   RTE_FLOW_ITEM_TYPE_PPPOES           |   PPPoE header                        |
|   RTE_FLOW_ITEM_TYPE_PPPOED           |   PPPoE header                        |
|   RTE_FLOW_ITEM_TYPE_PPPOE_PROTO_ID   |   PPPoE session protocol identifier   |
|   RTE_FLOW_ITEM_TYPE_END              |   End marker for item lists           |

##### Item ETH
|   Data Field  |   Value   |
|---------------|-----------|
|   dst         |   string  |
|   src         |   string  |
|   type        |   0-65535 |

An example of ETH Pattern Item:
```yaml
  - type: RTE_FLOW_ITEM_TYPE_IPV4
    spec:
      dst: 00:00:12:23:00:01
    last:
      dst: 00:00:12:23:00:1e
    mask:
      dst: ff:ff:ff:ff:ff:0
```
##### Item Vlan
|   Data Field  |   Value   |
|---------------|-----------|
|   tci         |  0-65535  |
|   inner_type  |  0-65535  |

An example of ETH Pattern Item:
```yaml
  - type: RTE_FLOW_ITEM_TYPE_VLAN
    spec:
      inner_type: 0x8100
```
##### Item IPv4
|   Data Field  |   Value   |
|---------------|-----------|
|   hdr         |   struct  |

###### Item IPv4 header

|   Data Field      |   Value   |
|-------------------|-----------|
|   version_ihl     |   0-255   |
|   type_of_service |   0-255   |
|   total_length    |   0-65535 |
|   packet_id       |   0-65535 |
|   fragment_offset |   0-65535 |
|   time_to_live    |   0-255   |
|   next_proto_id   |   0-255   |
|   hdr_checksum    |   0-65535 |
|   src_addr        |   string  |
|   dst_addr        |   string  |

An example of IPv4 Pattern Item:
```yaml
  - type: RTE_FLOW_ITEM_TYPE_IPV4
    spec:
      hdr:
        dst_addr: 192.168.10.9
    last:
      hdr:
        dst_addr: 192.168.10.99
    mask:
      hdr:
        dst_addr: 255.255.255.0
```
##### Item UDP
|   Data Field  |   Value   |
|---------------|-----------|
|   hdr         |   struct  |

###### Item UDP header
|   Data Field      |   Value   |
|-------------------|-----------|
|   src_port        |   0-65535 |
|   dst_port        |   0-65535 |
|   dgram_len       |   0-65535 |
|   dgram_cksum     |   0-65535 |

An example of UDP Pattern Item:
```yaml
  - type: RTE_FLOW_ITEM_TYPE_UDP
    spec:
      hdr:
        dst_port: 67
```
##### Item PPPOES/PPPOED
|   Data Field      |   Value   |
|-------------------|-----------|
|   version_type    |   0-255   |
|   code            |   0-255   |
|   session_id      |   0-65535 |
|   length          |   0-65535 |

An example of PPPOES Pattern Item:
```yaml
  - type: RTE_FLOW_ITEM_TYPE_PPPOES
    spec:
      version_type: 0x01
      code: 0x09
```
NOTE: A recent [Ice COMMS DDP package](https://downloadcenter.intel.com/download/29889/Intel-Ethernet-800-Series-Telecommunication-Comms-Dynamic-Device-Personalization-DDP-Package) needs to be loaded in order to create items of type PPPOES/PPPOED.

##### Item PPPOE PROTO ID
|   Data Field      |   Value   |
|-------------------|-----------|
|   proto_id        |   0-65535 |

An example of PPPOE PROTO ID Pattern Item:
```yaml
   - type: RTE_FLOW_ITEM_TYPE_PPPOE_PROTO_ID
     spec:
      proto_id: 0xc021
```
NOTE: A recent [Ice COMMS DDP package](https://downloadcenter.intel.com/download/29889/Intel-Ethernet-800-Series-Telecommunication-Comms-Dynamic-Device-Personalization-DDP-Package) needs to be loaded in order to create items of type PPPOE PROTO ID.
### Actions
Actions can alter the fate of matching traffic, its contents or properties. A list of actions can be assigned to a flow rule. These actions are performed in a given order and can require additional configuration. 

#### Supported Action Types

|   Action                                  |   Description                                                                     |
|-------------------------------------------|-----------------------------------------------------------------------------------|
|   RTE_FLOW_ACTION_TYPE_VF                 |   Direct matching traffic to a given virtual function of the current device       |
|   RTE_FLOW_ACTION_TYPE_VOID               |   Packets are ignored and simply discarded by PMDs                                |
|   RTE_FLOW_ACTION_TYPE_PASSTHRU           |   Make a flow rule non-terminating                                                |
|   RTE_FLOW_ACTION_TYPE_FLAG               |   Attach an integer flag value to packets                                         |
|   RTE_FLOW_ACTION_TYPE_DROP               |   Drop packets                                                                    |
|   RTE_FLOW_ACTION_TYPE_PF                 |   Direct matching traffic to the physical function (PF) of the current device     |
|   RTE_FLOW_ACTION_TYPE_OF_DEC_MPLS_TTL    |   Decrement MPLS TTL                                                              |
|   RTE_FLOW_ACTION_TYPE_OF_DEC_NW_TTL      |   Decrement IP TTL                                                                |
|   RTE_FLOW_ACTION_TYPE_OF_COPY_TTL_OUT    |   Copy TTL “outwards”                                                             |
|   RTE_FLOW_ACTION_TYPE_OF_COPY_TTL_IN     |   Copy TTL “inwards”                                                              |
|   RTE_FLOW_ACTION_TYPE_OF_POP_VLAN        |   Pop the outer VLAN tag                                                          |
|   RTE_FLOW_ACTION_TYPE_VXLAN_DECAP        |   Decapsulate by stripping all headers of the VXLAN tunnel network overlay        |
|   RTE_FLOW_ACTION_TYPE_NVGRE_DECAP        |   Decapsulate by stripping all headers of the NVGRE tunnel network overlay        |
|   RTE_FLOW_ACTION_TYPE_MAC_SWAP           |   Swap the source and destination MAC addresses in the outermost Ethernet header  |
|   RTE_FLOW_ACTION_TYPE_DEC_TTL            |   Decrease TTL value                                                              |
|   RTE_FLOW_ACTION_TYPE_INC_TCP_SEQ        |   Increase sequence number in the outermost TCP header                            |
|   RTE_FLOW_ACTION_TYPE_DEC_TCP_SEQ        |   Decrease sequence number in the outermost TCP header                            |
|   RTE_FLOW_ACTION_TYPE_INC_TCP_ACK        |   Increase acknowledgment number in the outermost TCP header                      |
|   RTE_FLOW_ACTION_TYPE_DEC_TCP_ACK        |   Decrease acknowledgment number in the outermost TCP header                      |
|   RTE_FLOW_ACTION_TYPE_END                |   End marker for action lists                                                     |

#### Action VF
|   Data Field  |   Value           |
|---------------|-------------------|
|   Reserved    |   0               |
|   Original    |   0-1             |
|   Id          |   0-255           |

An example of Action VF:
```yaml
  - type: RTE_FLOW_ACTION_TYPE_VF
    conf:
      id: 1
```

NOTE: At the moment only Action of type VF has additional config. Other actions have no configurable properties.

### Attributes
Attributes are the additional properties of a flow rule.

| Attribute | Description                                                           | Value        |
|-----------|-----------------------------------------------------------------------|--------------|
| group     | Group similar rules                                                   | 0-4294967295 |
| priority  | Flow rule priority level                                              | 0-4294967295 |
|	ingress   | Apply flowrule to inbound traffic                                     | 0-1          |
|	egress    | Apply flowrule to outbound traffic                                    | 0-1          |
|	transfer  | Transfer a flow rule to the lowest possible level of device endpoints | 0-1          |