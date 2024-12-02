---
title: "switch"
asciinema: true
---

# switch

## Description

The switch filter is used to forward traffic to another flow along the requested path, to facilitate cross-cluster operations. No alternation is required for Elasticsearch clusters, and all APIs in each cluster can be accessed, including APIs used for index read/write and cluster operations.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: es1-flow
    filter:
      - elasticsearch:
          elasticsearch: es1
  - name: es2-flow
    filter:
      - elasticsearch:
          elasticsearch: es2
  - name: cross_cluste_search
    filter:
      - switch:
          path_rules:
            - prefix: "es1:"
              flow: es1-flow
            - prefix: "es2:"
              flow: es2-flow
      - elasticsearch:
          elasticsearch: dev  #elasticsearch configure reference name
```

In the above example, the index beginning with `es1:` is forwarded to the `es1` cluster, the index beginning with `es2:` is forwarded to the `es2` cluster, and unmatched indexes are forwarded to the `dev` cluster. Clusters of different versions can be controlled within one Kibana. See the following example.

```
# GET es1:_cluster/health
{
  "cluster_name" : "elasticsearch",
  "status" : "yellow",
  "timed_out" : false,
  "number_of_nodes" : 1,
  "number_of_data_nodes" : 1,
  "active_primary_shards" : 37,
  "active_shards" : 37,
  "relocating_shards" : 0,
  "initializing_shards" : 0,
  "unassigned_shards" : 9,
  "delayed_unassigned_shards" : 0,
  "number_of_pending_tasks" : 0,
  "number_of_in_flight_fetch" : 0,
  "task_max_waiting_in_queue_millis" : 0,
  "active_shards_percent_as_number" : 80.43478260869566
}

# GET es2:_cluster/health
{
  "cluster_name" : "elasticsearch",
  "status" : "yellow",
  "timed_out" : false,
  "number_of_nodes" : 1,
  "number_of_data_nodes" : 1,
  "active_primary_shards" : 6,
  "active_shards" : 6,
  "relocating_shards" : 0,
  "initializing_shards" : 0,
  "unassigned_shards" : 6,
  "delayed_unassigned_shards" : 0,
  "number_of_pending_tasks" : 0,
  "number_of_in_flight_fetch" : 0,
  "task_max_waiting_in_queue_millis" : 0,
  "active_shards_percent_as_number" : 50.0
}
```

You can run commands to achieve the same effect.

```
root@infini:/opt/gateway# curl -v  192.168.3.4:8000/es1:_cat/nodes
*   Trying 192.168.3.4...
* TCP_NODELAY set
* Connected to 192.168.3.4 (192.168.3.4) port 8000 (#0)
> GET /es1:_cat/nodes HTTP/1.1
> Host: 192.168.3.4:8000
> User-Agent: curl/7.58.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: INFINI
< Date: Thu, 14 Oct 2021 10:37:39 GMT
< content-type: text/plain; charset=UTF-8
< Content-Length: 45
< X-Backend-Cluster: dev1
< X-Backend-Server: 192.168.3.188:9299
< X-Filters: filters->switch->filters->elasticsearch->skipped
<
192.168.3.188 48 38 5    cdhilmrstw * LENOVO
* Connection #0 to host 192.168.3.4 left intact
root@infini:/opt/gateway# curl -v  192.168.3.4:8000/es2:_cat/nodes
*   Trying 192.168.3.4...
* TCP_NODELAY set
* Connected to 192.168.3.4 (192.168.3.4) port 8000 (#0)
> GET /es2:_cat/nodes HTTP/1.1
> Host: 192.168.3.4:8000
> User-Agent: curl/7.58.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Server: INFINI
< Date: Thu, 14 Oct 2021 10:37:48 GMT
< content-type: text/plain; charset=UTF-8
< Content-Length: 146
< X-elastic-product: Elasticsearch
< Warning: 299 Elasticsearch-7.14.0-dd5a0a2acaa2045ff9624f3729fc8a6f40835aa1 "Elasticsearch built-in security features are not enabled. Without authentication, your cluster could be accessible to anyone. See https://www.elastic.co/guide/en/elasticsearch/reference/7.14/security-minimal-setup.html to enable security."
< X-Backend-Cluster: dev
< X-Backend-Server: 192.168.3.188:9216
< X-Filters: filters->switch->filters->elasticsearch->skipped
<
192.168.3.188 26 38 3    cdfhilmrstw - node-714-1
192.168.3.188 45 38 3    cdfhilmrstw * LENOVO
192.168.3.188 43 38 4    cdfhilmrstw - node-714-2
* Connection #0 to host 192.168.3.4 left intact
```

## Parameter Description

| Name              | Type   | Description                                                                                                                                                                                    |
| ----------------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| path_rules        | array  | Matching rule based on the URL                                                                                                                                                                 |
| path_rules.prefix | string | Prefix string for matching. It is recommended that the prefix string end with `:`. After matching, the URL prefix is removed from the traffic, which is then forwarded to the subsequent flow. |
| path_rules.flow   | string | Name of the flow for processing a matched request                                                                                                                                              |
| remove_prefix     | bool   | Whether to remove matched prefix string before request forwarding. The default value is `true`.                                                                                                |
| continue          | bool   | Whether to continue the flow after hit. Request returns immediately after it is set to `false`. The default value is `false`.                                                                  |
| unescape          | bool   | Whether to unescape the url path. The default value is `true`.                                                                                                                                 |
