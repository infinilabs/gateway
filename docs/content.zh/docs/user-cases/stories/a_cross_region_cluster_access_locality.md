---
title: "作业帮跨云集群的就近本地访问"
weight: 30
draft: false
author: 天降大任@作业帮
---

# 跨云集群的就近本地访问

## 业务需求

作业帮为了确保某个业务 Elasticsearch 集群的高可用，在百度云和华为云上面采取了双云部署，即将单个 Elasticsearch 集群跨云进行部署，并且要求业务请求优先访问本地云。

## Elasticsearch 单集群双云实现

Elasticsearch 集群采用 Master 与 Data 节点分离的架构。 目前主力云放 2 个 Master，另外一个云放一个 Master。 主要考虑就是基础设施故障中，专线故障问题是大多数，某个云厂商整体挂的情况基本没有。
所以设置了主力云，当专线故障时，主力云的 Elasticsearch 是可以读写的，业务把流量切到主力云就行了。

具体配置方式如下。

首先，在 Master 节点上设置：

```
cluster.routing.allocation.awareness.attributes: zone_id
cluster.routing.allocation.awareness.force.zone_id.values: zone_baidu,zone_huawei
```

然后分别在百度云上数据节点上设置：

```
node.attr.zone_id: zone_baidu
```

和华为云上数据节点上设置：

```
node.attr.zone_id: zone_huawei
```

创建索引采用 1 副本，可以保证百度云与华为云上都有一份相同的数据。

业务访问方式如下图：

{{% load-img "/img/cross_region_cluster.jpg" "跨云单集群就近访问" %}}

- 百度云业务 -> 百度 lb -> INFINI Gateway (百度) -> Elasticsearch （百度云 data 节点）
- 华为云业务 -> 华为 lb -> INFINI Gateway (华为) -> Elasticsearch （华为云 data 节点）

## 极限网关配置

Elasticsearch 支持一个 [Preference](https://www.elastic.co/guide/en/elasticsearch/reference/master/search-search.html#search-preference) 参数来设置请求的优先访问，通过在两个云内部的极限网关分别设置各自请求默认的 Preference 参数，让各个云内部的请求优先发往本云内的数据节点，即可实现请求的就近访问。

具体的百度云的 INFINI Gateway 配置如下（华为云大体相同，就不重复贴了）：

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
          - preference -> _prefer_nodes:node-id-of-data-baidu01,node-id-of-data-baidu02 #通过配置preference的_prefer_nodes为所有的百度data节点的node_id，来实现百度云的业务优先访问百度云的节点，最大程度避免跨云访问，对业务更友好。
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
            - data #配置为data，请求只发送到data节点
        tags:
          include:
            - zone_id: zone_baidu #只转发给百度云里面的节点


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

## 总结与收益

### 引入极限网关前故障回顾

百度云业务访问 Elasticsearch 集群，拉取每天的增量数据同步到 Hive 集群，其中有几个任务失败后，又重新同步。结果是部分数据从华为云的 Elasticsearch 节点拉取到百度云的 Hive 集群中，数据量巨大导致跨云专线流量监控告警。由于线上业务、MySQL、Redis、Elasticsearch 等使用同一根专线，
此次故障影响面较大。临时解决方案是业务修改语句加入 Preference 参数来实现业务只拉取本地云数据，减少对专线的占用。但是一方面业务改造及维护成本较高；另一方面作为 DBA 会担心业务改造有疏漏、新增业务遗忘 Preference 参数、以及后期调整成本较高，这始终是一个风险点。

### 引入极限网关的收益

在原有架构上加入极限网关，可以在业务不修改代码的情况下做到优先访问本地云，提升访问速度的同时，最大限度减少对专线的压力。

> 作者：赵青，前网易 DBA，工作主要涉及 Oracle、MySQL、Redis、Elasticsearch、Tidb、OB 等组件的运维以及运维自动化、平台化、智能化等工作。现就职于作业帮。
