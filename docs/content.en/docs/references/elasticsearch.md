---
title: "Elasticsearch"
weight: 47
---

# Elasticsearch

## Defining a Resource

INFINI Gateway supports multi-cluster access and different versions. Each cluster serves as one Elasticsearch back-end resource and can be subsequently used by INFINI Gateway in multiple locations. See the following example.

```
elasticsearch:
- name: local
  enabled: true
  endpoint: https://127.0.0.1:9200
- name: dev
  enabled: true
  endpoint: https://192.168.3.98:9200
  basic_auth:
    username: elastic
    password: pass
- name: prod
  enabled: true
  endpoint: http://192.168.3.201:9200
  discovery:
    enabled: true
    refresh:
      enabled: true
      interval: 10s
  basic_auth:
    username: elastic
    password: pass
```

The above example defines a local development test cluster named `local` and a development cluster named `dev`.
Authentication is enabled in the development cluster, in which corresponding usernames and passwords are also defined. In addition, one production cluster named `prod` is defined, and the auto node topology discovery and update of the cluster are enabled through the `discovery` parameter.

## Parameter Description

| Name                                    | Type   | Description                                                                                                                     |
| --------------------------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------- | --- |
| name                                    | string | Name of an Elasticsearch cluster                                                                                                |
| project                                 | string | project name                                                                                                                    |
| location.provider                       | string | the service provider region info of the cluster                                                                                 |
| location.region                         | string | the region info of the cluster                                                                                                  |
| location.dc                             | string | the data center info of the cluster                                                                                             |
| location.rack                           | string | the rack info of the cluster                                                                                                    |
| labels                                  | map    | cluster labels                                                                                                                  |
| tags                                    | array  | cluster tags                                                                                                                    |
| enabled                                 | bool   | Whether the cluster is enabled                                                                                                  |
| endpoint                                | string | Elasticsearch access address, for example, `http://localhost:9200`                                                              |
| endpoints                               | array  | List of Elasticsearch access addresses. Multiple entry addresses are supported for redundancy.                                  |
| schema                                  | string | Protocol type: `http` or `https`                                                                                                |
| host                                    | string | Elasticsearch host, in the format of `localhost:9200`. Either the host or endpoint configuration mode can be used.              |
| hosts                                   | array  | Elasticsearch host list. Multiple entry addresses are supported for redundancy.                                                 |
| request_timeout                         | int    | Request timeout duration, in seconds, default `30`                                                                              |
| request_compress                        | bool   | Whether to enable Gzip compression                                                                                              |
| basic_auth                              | object | Authentication information                                                                                                      |
| basic_auth.username                     | string | Username                                                                                                                        |
| basic_auth.password                     | string | Password                                                                                                                        |
| discovery                               | object | Cluster discovery settings                                                                                                      |
| discovery.enabled                       | bool   | Whether to enable cluster topology discovery                                                                                    |
| discovery.refresh                       | object | Cluster topology update settings                                                                                                |
| discovery.refresh.enabled               | bool   | Whether to enable auto cluster topology update                                                                                  |
| discovery.refresh.interval              | string | Interval of auto cluster topology update                                                                                        |
| traffic_control                         | object | Node-level overall traffic control of the cluster                                                                               |
| traffic_control.enabled      | bool    | Whether to enabled traffic control                                                                      |
| traffic_control.max_bytes_per_node      | int    | Maximum allowable number of request bytes per second                                                                            |
| traffic_control.max_qps_per_node        | int    | Maximum allowable number of requests per second, regardless of read or write requests                                           |
| traffic_control.max_connection_per_node | int    | Maximum allowable number of connections per node                                                                                |
| traffic_control.max_wait_time_in_ms     | int    | In case of throttled, the maximum allowable waiting time in ms, the default is `10000`                                          |     |
| allow_access_when_master_not_found      | bool   | Still allow access to visit this elasticsearch when it is in error `master_not_discovered_exception` , default value is `false` |
