---
title: "Nearest Cluster Access Across Two Cloud Providers"
weight: 30
draft: false
author:
---

# Nearest Cluster Access Across Two Cloud Providers

## Service Requirements

To ensure the high availability of the Elasticsearch service, Zuoyebang deploys a single Elasticsearch cluster on both Baidu Cloud and Huawei Cloud and requires that service requests be sent to the nearest cloud first.

## Deployment of a Single Elasticsearch Cluster on Dual Clouds

The Elasticsearch cluster uses an architecture with master nodes separated from data nodes. Currently, the main cloud is used to accommodate two master nodes and the other cloud is used to accommodate another master node.
The main consideration is that infrastructure failures are mostly dedicated line failures, and the overall breakdown of a provider's cloud rarely occurs. Therefore, the main cloud is configured. When a dedicated line failure occurs, the Elasticsearch cluster on the main cloud is read/write and service traffic can be switched to the main cloud.

The configuration is as follows:

First, complete the following settings on the master nodes:

```
cluster.routing.allocation.awareness.attributes: zone_id
cluster.routing.allocation.awareness.force.zone_id.values: zone_baidu,zone_huawei
```

Then, perform the following settings on data nodes on Baidu Cloud:

```
node.attr.zone_id: zone_baidu
```

Perform the following settings on data nodes on Huawei Cloud:

```
node.attr.zone_id: zone_huawei
```

Indexes are created using one copy, which can ensure that the same copy of data exists on Baidu Cloud and Huawei Cloud.

The service access mode is shown as follows:

{{% load-img "/img/cross_region_cluster.jpg" "Nearest Access to a Single Cluster Across Clouds" %}}

- Baidu Cloud service -> Baidu lb -> INFINI Gateway (Baidu Cloud) -> Elasticsearch (data nodes on Baidu Cloud)
- Huawei Cloud service -> Huawei lb -> INFINI Gateway (Huawei Cloud) -> Elasticsearch (data nodes on Huawei Cloud)

## Configuring INFINI Gateway

Elasticsearch uses the [Preference](https://www.elastic.co/guide/en/elasticsearch/reference/master/search-search.html#search-preference) parameter to set the request access priority. Set the default Preference parameter for requests on INFINI Gateway inside the two clouds so that requests inside each cloud are sent to data nodes in the local cloud first, thereby implementing nearest access of requests.

The specific configuration on INFINI Gateway inside Baidu Cloud is as follows (the configuration on INFINI Gateway inside Huawei Cloud is basically the same and is not provided here):

```
path.data: data
path.logs: log

entry:
- name: es-test
  enabled: true
  router: default
  network:
      binding: 0.0.0.0:9200
      reuse_port: true

router:
- name: default
  default_flow: es-test

flow:
- name: es-test
  filter:
    - set_request_query_args:
        args:
          - preference -> _prefer_nodes:node-id-of-data-baidu01,node-id-of-data-baidu02 #Set _prefer_nodes of Preference to all Baidu data nodes(use `node_id` of these nodes) so that Baidu Cloud service accesses the nodes of Baidu Cloud first, thereby avoiding cross-cloud access to the maximum extent and enabling the service to run more smoothly.
        when:
          contains:
            _ctx.request.path: /_search
    - elasticsearch:
        elasticsearch: default
        refresh:
            enabled: true
            interval: 10s
        roles:
            include:
            - data #Set it to data so that requests are sent only to the data node.
        tags:
          include:
            - zone_id: zone_baidu #Requests are forwarded only to nodes in Baidu Cloud.


elasticsearch:
- name: default
  enabled: true
  endpoint: http://10.10.10.10:9200
  allow_access_when_master_not_found: true
  discovery:
      enabled: true
  refresh:
      enabled: true
      interval: 10s
  basic_auth:
      username: elastic
      password: elastic
```

## Summary and Benefits

### Retrospect of Failures Before INFINI Gateway Is Introduced

When Baidu Cloud service accesses the Elasticsearch cluster, it pulls daily incremental data from the Hive cluster and synchronizes it to Elasticsearch. Some tasks may fail and data is synchronized again. As a result, some data is pulled from the Elasticsearch node inside Huawei Cloud to the Hive cluster of Baidu Cloud. The huge amount of data triggers an alarm about cross-cloud dedicated line traffic monitoring. Online services, MySQL, Redis, and Elasticsearch use the same dedicated line.
The impact of the failures is huge. The temporary solution is to add the Preference parameter to the service modification statement so that the services only pull local cloud data, reducing the occupancy of the dedicated line. The service transformation and maintenance costs are high. In addition, DBA has worries that there are omissions in service transformation, the Preference parameter is ignored for new services, and later adjustment costs are high. These are always risk points.

### Benefits of INFINI Gateway

After INFINI Gateway is added to the original architecture, services can preferentially access the local cloud with the service code not modified. In this way, CPU resources of the server are fully utilized and all CPU resources of each node are used.

> Author: Zhao Qing, former DBA of NetEase, mainly involved in the O&M of Oracle, MySQL, Redis, Elasticsearch, Tidb, OB, and other components, as well as O&M automation, platform-based application, and intelligence. Now he is working in Zuoyebang.
