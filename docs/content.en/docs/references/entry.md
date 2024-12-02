---
title: "Service Entry"
weight: 20
---

# Service Entry

## Defining an Entry

Each gateway must expose at least one service entrance to receive operation requests of services. In INFINI Gateway, the service entrance is called an `entry`, which can be defined using the following parameters:

```
entry:
  - name: es_gateway
    enabled: true
    router: default
    network:
      binding: 0.0.0.0:8000
      reuse_port: true
    tls:
      enabled: false
```

The `network.binding` parameter can be used to specify the IP address and port to be bound and listened to after the service is started. INFINI Gateway supports port reuse, that is, multiple INFINI Gateways can share the same IP address and port.
In this way, server resources can be fully utilized and the configuration of different gateway processes can be modified dynamically (you can start multiple processes, and then restart the processes in sequence after modifying the configuration), without interrupting normal client requests.

For each request sent to the `entry`, requested traffic is routed by `router`. Rules are defined for `router` separately so that the rules are used in different `entry` settings. In `entry`, the `router` parameter can be used to specify the `router` rules to be used and `default` is defined here.

## TLS Configuration

TLS transmission encryption can be seamlessly enabled on INFINI Gateway. You can switch to HTTPS communication mode by setting `tls.enabled` to `true`. INFINI Gateway can automatically generate certification files.

INFINI Gateway also allows you to define the path of the certification file. The configuration is as follows:

```
entry:
  - name: es_gateway
    enabled: true
    router: default
    network:
      binding: 0.0.0.0:8000
      reuse_port: true
    tls:
      enabled: true
      cert_file: /etc/ssl.crt
      key_file: /etc/ssl.key
      skip_insecure_verify: false
```

## Multiple Services

INFINI Gateway can listen on multiple service entries at the same time. The listened address, protocol, and router of each service entry can be separately defined to meet different service requirements. The following shows a configuration example.

```
entry:
  - name: es_ingest
    enabled: true
    router: ingest_router
    network:
      binding: 0.0.0.0:8000
  - name: es_search
    enabled: true
    router: search_router
    network:
      binding: 0.0.0.0:9000
```

The above example defines a service entry named `es_ingest` to listen on the address `0.0.0.0:8000`, and all requests are processed through `ingest_router`.
In the example, one `es_search` service is also defined, the listening port is `9000`, and `search_router` is used for request processing to implement read/write separation of services.
In addition, different service entries can be defined for different back-end Elasticsearch clusters, and the gateway can forward requests as a proxy.

## IPv6 Support

INFINI Gateway support to binding to IPv6 address，for example:

```
entry:
  - name: es_ingest
    enabled: true
    router: ingest_router
    network:
#      binding: "[ff80::4e2:7fb6:7db6:a839%en0]:8000"
      binding: "[::]:8000"
```

## Parameter Description

| Name                       | Type   | Description                                                                          |
| -------------------------- | ------ | ------------------------------------------------------------------------------------ |
| name                       | string | Name of a service entry                                                              |
| enabled                    | bool   | Whether the entry is enabled                                                         |
| max_concurrency            | int    | Maximum concurrency connection number, which is `10000` by default.                  |
| router                     | string | Router name                                                                          |
| network                    | object | Relevant network configuration                                                       |
| tls                        | object | TLS secure transmission configuration                                                |
| network.host               | string | Network address listened to by the service, for example, `192.168.3.10`              |
| network.port               | int    | Port address listened to by the service, for example, `8000`                         |
| network.binding            | string | Network binding address listened to by the service, for example, `0.0.0.0:8000`      |
| network.publish            | string | External access address listened to by the service, for example, `192.168.3.10:8000` |
| network.reuse_port         | bool   | Whether to reuse the network port for multi-process port sharing                     |
| network.skip_occupied_port | bool   | Whether to automatically skip occupied ports                                         |
| tls.enabled                | bool   | Whether TLS secure transmission is enabled                                           |
| tls.cert_file              | string | Path to the public key of the TLS security certificate                               |
| tls.key_file               | string | Path to the private key of the TLS security certificate                              |
| tls.skip_insecure_verify   | bool   | Whether to ignore TLS certificate verification                                       |
