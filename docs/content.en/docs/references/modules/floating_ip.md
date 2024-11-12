---
title: "Floating IP"
weight: 20
draft: false
---

# Floating IP

The embedded floating IP feature of INFINI Gateway can implement dual-node hot standby and failover. INFINI Gateway innately provides high availability for L4 network traffic, and no extra software and devices are required to prevent proxy service interruption caused by downtime or network failures.

{{< hint info >}}
Note:

- This feature supports only Mac OS and Linux OS. The gateway must run as the user root.
- This feature relies on the `ping` and `ifconfig` commands of the target system. Therefore, ensure that related packages are installed by default.
- The network interface cards (NICs) of a group of gateways, on which floating IP is enabled, should belong to the same subnet, and devices on the Intranet can communicate with each other in broadcast mode (the actual IP address and floating IP address of a gateway need to be different only in the last bit, for example, `192.168.3.x`).
  {{< /hint >}}

## Function Demonstration

- [Youtube](https://youtu.be/-RUhkBcm4fc)
- [Bilibili](https://www.bilibili.com/video/BV1DK4y1V7um/)

## What Is a Floating IP?

INFINI Gateway achieves high availability by using a floating IP, which is also called a virtual IP or dynamic IP.
Each server must have an IP address for communication and the IP address of a server is usually static and allocated in advance.
If the server malfunctions, the IP address and the services deployed on the server are inaccessible. A floating IP address is usually a public and routable IP address that is not automatically allocated to a physical device.
The project manager can temporarily allocate this dynamic IP address to one or more physical devices. The physical devices have automatically assigned static IP addresses for communicating with devices on the Intranet. This Intranet uses private addresses that are not routable. Services of physical devices on the Intranet can be identified and accessed by external networks only through the floating IP address.

{{% load-img "/img/floating-ip.jpg" "" %}}

## Why Is a Floating IP Needed?

One typical floating IP switching scenario is that, when a device bound with a floating IP address malfunctions, the floating IP address floats to another device on the network.
The new device immediately replaces the faulty device to provide services externally. This creates high availability for network services. For service consumers, only the floating IP needs to be specified.
Floating IPs are very useful. In certain scenarios, for example, only one service IP address is allowed for the client or SDK, which means that the IP address must be highly available. INFINI Gateway can effectively solve this problem. When two independent INFINI Gateway servers are used, you are advised to deploy them on independent physical servers.
The two INFINI Gateways work in dual-node hot standby mode. If any of the gateways malfunction, front-end services can still be accessed.

## Enabling Floating IP

To enable the floating IP feature of INFINI Gateway, modify the `gateway.yml` configuration file by adding the following configuration:

```
floating_ip:
  enabled: true
```

INFINI Gateway can automatically detect NIC device information and bind the virtual IP address to the Intranet communication port. It is very intelligent and easy to use. By default, the IP address to be listened to is `*.*.*.234` in the network segment, to which the machine belongs.
Assume that the physical IP address of the machine is `192.168.3.35`. The default floating IP address is `192.168.3.234`. This default IP address is only used to facilitate configuration and quick startup. If you need to use a user-defined floating IP address, supplement complete parameters.

## Related Parameter Settings

The following is an example of configuration parameters about floating IP:

```
floating_ip:
  enabled: true
  ip: 192.168.3.234
  netmask: 255.255.255.0
  interface: en1
```

The parameters are described as follows:

| Name                      | Type   | Description                                                                                                                                                                                                                                         |
| ------------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `enabled`                 | bool   | Whether floating IP is enabled, which is set to `false` by default.                                                                                                                                                                                 |
| `interface`               | string | NIC device name. If this parameter is not specified, the name of the first device that listens to the first non-local address is selected. If a server has multiple NIC cards, you are advised to manually set this parameter.                      |
| `ip`                      | string | Listened floating IP address, which is `*.*.*.234` in the network segment, to which the current physical NIC belongs. You are advised to manually set the floating IP address. The floating IP address cannot conflict with an existing IP address. |
| `local_ip`                | string | The physical IP address                                                                                                                                                                                                                             |
| `netmask`                 | string | Subnet mask of the floating IP address, which is the subnet mask of the NIC or `255.255.255.0` by default.                                                                                                                                          |
| `echo.port`               | int    | The ports between gateway nodes for heartbeat detection, make sure that connect to this port are allowed, default `61111`                                                                                                                           |
| `echo.dial_timeout_in_ms` | int    | Timeout for heartbeat detection dialing, default `10000`                                                                                                                                                                                            |
| `echo.timeout_in_ms`      | int    | Timeout for heartbeat detection, default `10000`                                                                                                                                                                                                    |
