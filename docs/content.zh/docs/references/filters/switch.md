---
title: "switch"
asciinema: true
---

# switch

## 描述

switch 过滤器用来将流量按照请求路径转发到另外的一个处理流程，可以方便的实现跨集群操作，且 Elasticsearch 集群不需要做任何修改，且各个集群内所有的 API 都可以访问，包括索引的读写和集群操作。

## 配置示例

一个简单的示例如下：

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

上面的例子中，以 `es1:` 开头的索引将转发给集群 `es1` 集群，以 `es2:` 开头的索引转发给 `es2` 集群，不匹配的转发给 `dev` 集群，在一个 Kibana 里面可以直接操作不同版本的集群了，如下：

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

通过命令行也同样可以：

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

## 参数说明

| 名称              | 类型   | 说明                                                                                              |
| ----------------- | ------ | ------------------------------------------------------------------------------------------------- |
| path_rules        | array  | 根据 URL 路径的匹配规则                                                                           |
| path_rules.prefix | string | 匹配的不包含 `/`开头的前缀字符串，，建议以 `:` 结尾，匹配之后会移除该 URL 前缀转发给后面的 flow。 |
| path_rules.flow   | string | 匹配之后用于处理该请求的 flow 名称。                                                              |
| remove_prefix     | bool   | 转发请求之前，是否移除前缀匹配上的字符串，默认 `true`                                             |
| continue          | bool   | 匹配跳转之后，是否还继续执行后面的流程，设置成 `false` 则立即返回，默认 `false`。                 |
| unescape          | bool   | 是否对 path 参数进行 URL Decode 解码，默认 `true`.                                                |
