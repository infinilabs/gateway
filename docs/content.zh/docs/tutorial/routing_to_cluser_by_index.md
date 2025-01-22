---
title: "在 Kibana 里统一访问来自不同集群的索引"
weight: 100
---

# 在 Kibana 里统一访问来自不同集群的索引

现在有这么一个需求，客户根据需要将数据按照业务维度划分，将索引分别存放在了不同的三个集群，
将一个大集群拆分成多个小集群有很多好处，比如降低了耦合，带来了集群可用性和稳定性方面的好处，也避免了单个业务的热点访问造成其他业务的影响，
尽管拆分集群是很常见的玩法，但是管理起来不是那么方便了，尤其是在查询的时候，可能要分别访问三套集群各自的 API，甚至要切换三套不同的 Kibana 来访问集群的数据，
那么有没有办法将他们无缝的联合在一起呢？

## 极限网关!

答案自然是有的，通过将 Kibana 访问 Elasticsearch 的地址切换为极限网关的地址，我们可以将请求按照索引来进行智能的路由，
也就是当访问不同的业务索引时会智能的路由到不同的集群，如下图：

{{% load-img "/img/smart_route_by_index.png" "" %}}

上图，我们分别有 3 个不同的索引：

- apm-\*
- erp-\*
- mall-\*

分别对应不同的三套 Elasticsearch 集群:

- ES1-APM
- ES2-ERP
- ES3-MALL

接下来我们来看如何在极限网关里面进行相应的配置来满足这个业务需求。

## 配置集群信息

首先配置 3 个集群的连接信息。

```
elasticsearch:
  - name: es1-apm
    enabled: true
    endpoints:
     - http://192.168.3.188:9206
  - name: es2-erp
    enabled: true
    endpoints:
     - http://192.168.3.188:9207
  - name: es3-mall
    enabled: true
    endpoints:
     - http://192.168.3.188:9208
```

## 配置服务 Flow

然后，我们定义 3 个 Flow，分别对应用来访问 3 个不同的 Elasticsearch 集群，如下：

```
flow:
  - name: es1-flow
    filter:
      - elasticsearch:
          elasticsearch: es1-apm
  - name: es2-flow
    filter:
      - elasticsearch:
          elasticsearch: es2-erp
  - name: es3-flow
    filter:
      - elasticsearch:
          elasticsearch: es3-mall
```

然后再定义一个 flow 用来进行路径的判断和转发，如下：

```
  - name: default-flow
    filter:
      - switch:
          remove_prefix: false
          path_rules:
            - prefix: "apm-"
              flow: es1-flow
            - prefix: "erp-"
              flow: es2-flow
            - prefix: "mall-"
              flow: es3-flow
      - flow: #default flow
          flows:
            - es1-flow
```

根据请求路径里面的索引前缀来匹配不同的索引，并转发到不同的 Flow。

## 配置路由信息

接下来，我们定义路由信息，具体配置如下：

```
router:
  - name: my_router
    default_flow: default-flow
```

指向上面定义的默认 flow 来统一请求的处理。

## 定义服务及关联路由

最后，我们定义一个监听为 8000 端口的服务，用来提供给 Kibana 来进行统一的入口访问，如下：

```
entry:
  - name: es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8000
```

## 完整配置

最后的完整配置如下：

```
path.data: data
path.logs: log

entry:
  - name: es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8000

flow:
  - name: default-flow
    filter:
      - switch:
          remove_prefix: false
          path_rules:
            - prefix: "apm-"
              flow: es1-flow
            - prefix: "erp-"
              flow: es2-flow
            - prefix: "mall-"
              flow: es3-flow
      - flow: #default flow
          flows:
            - es1-flow
  - name: es1-flow
    filter:
      - elasticsearch:
          elasticsearch: es1-apm
  - name: es2-flow
    filter:
      - elasticsearch:
          elasticsearch: es2-erp
  - name: es3-flow
    filter:
      - elasticsearch:
          elasticsearch: es3-mall

router:
  - name: my_router
    default_flow: default-flow

elasticsearch:
  - name: es1-apm
    enabled: true
    endpoints:
     - http://192.168.3.188:9206
  - name: es2-erp
    enabled: true
    endpoints:
     - http://192.168.3.188:9207
  - name: es3-mall
    enabled: true
    endpoints:
     - http://192.168.3.188:9208
```

## 启动网关

直接启动网关，如下：

```
➜  gateway git:(master) ✗ ./bin/gateway -config sample-configs/elasticsearch-route-by-index.yml

   ___   _   _____  __  __    __  _
  / _ \ /_\ /__   \/__\/ / /\ \ \/_\ /\_/\
 / /_\///_\\  / /\/_\  \ \/  \/ //_\\\_ _/
/ /_\\/  _  \/ / //__   \  /\  /  _  \/ \
\____/\_/ \_/\/  \__/    \/  \/\_/ \_/\_/

[GATEWAY] A light-weight, powerful and high-performance elasticsearch gateway.
[GATEWAY] 1.0.0_SNAPSHOT, 2022-04-20 08:23:56, 2023-12-31 10:10:10, 51650a5c3d6aaa436f3c8a8828ea74894c3524b9
[04-21 13:41:21] [INF] [app.go:174] initializing gateway.
[04-21 13:41:21] [INF] [app.go:175] using config: /Users/medcl/go/src/infini.sh/gateway/sample-configs/elasticsearch-route-by-index.yml.
[04-21 13:41:21] [INF] [instance.go:72] workspace: /Users/medcl/go/src/infini.sh/gateway/data/gateway/nodes/c9bpg0ai4h931o4ngs3g
[04-21 13:41:21] [INF] [app.go:283] gateway is up and running now.
[04-21 13:41:21] [INF] [api.go:262] api listen at: http://0.0.0.0:2900
[04-21 13:41:21] [INF] [reverseproxy.go:255] elasticsearch [es1-apm] hosts: [] => [192.168.3.188:9206]
[04-21 13:41:21] [INF] [reverseproxy.go:255] elasticsearch [es2-erp] hosts: [] => [192.168.3.188:9207]
[04-21 13:41:21] [INF] [reverseproxy.go:255] elasticsearch [es3-mall] hosts: [] => [192.168.3.188:9208]
[04-21 13:41:21] [INF] [actions.go:349] elasticsearch [es2-erp] is available
[04-21 13:41:21] [INF] [actions.go:349] elasticsearch [es1-apm] is available
[04-21 13:41:21] [INF] [entry.go:312] entry [es_entry] listen at: http://0.0.0.0:8000
[04-21 13:41:21] [INF] [module.go:116] all modules are started
[04-21 13:41:21] [INF] [actions.go:349] elasticsearch [es3-mall] is available
[04-21 13:41:55] [INF] [reverseproxy.go:255] elasticsearch [es1-apm] hosts: [] => [192.168.3.188:9206]
```

网关启动成功之后，就可以通过网关的 IP+8000 端口来访问目标 Elasticsearch 集群了。

## 测试访问

首先通过 API 来访问测试一下，如下：

```
➜  ~ curl http://localhost:8000/apm-2022/_search -v
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 8000 (#0)
> GET /apm-2022/_search HTTP/1.1
> Host: localhost:8000
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 21 Apr 2022 05:45:44 GMT
< content-type: application/json; charset=UTF-8
< Content-Length: 162
< X-elastic-product: Elasticsearch
< X-Backend-Cluster: es1-apm
< X-Backend-Server: 192.168.3.188:9206
< X-Filters: filters->elasticsearch
<
* Connection #0 to host localhost left intact
{"took":142,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":0,"relation":"eq"},"max_score":null,"hits":[]}}%
```

可以看到 apm-2022 指向了后端的 `es1-apm` 集群。

继续测试，erp 索引的访问，如下：

```
➜  ~ curl http://localhost:8000/erp-2022/_search -v
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 8000 (#0)
> GET /erp-2022/_search HTTP/1.1
> Host: localhost:8000
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 21 Apr 2022 06:24:46 GMT
< content-type: application/json; charset=UTF-8
< Content-Length: 161
< X-Backend-Cluster: es2-erp
< X-Backend-Server: 192.168.3.188:9207
< X-Filters: filters->switch->filters->elasticsearch->skipped
<
* Connection #0 to host localhost left intact
{"took":12,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":0,"relation":"eq"},"max_score":null,"hits":[]}}%
```

继续测试，mall 索引的访问，如下：

```
➜  ~ curl http://localhost:8000/mall-2022/_search -v
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 8000 (#0)
> GET /mall-2022/_search HTTP/1.1
> Host: localhost:8000
> User-Agent: curl/7.54.0
> Accept: */*
>
< HTTP/1.1 200 OK
< Date: Thu, 21 Apr 2022 06:25:08 GMT
< content-type: application/json; charset=UTF-8
< Content-Length: 134
< X-Backend-Cluster: es3-mall
< X-Backend-Server: 192.168.3.188:9208
< X-Filters: filters->switch->filters->elasticsearch->skipped
<
* Connection #0 to host localhost left intact
{"took":8,"timed_out":false,"_shards":{"total":5,"successful":5,"skipped":0,"failed":0},"hits":{"total":0,"max_score":null,"hits":[]}}%
```

完美转发。

## 其他方式

除了使用 `switch` 过滤器，使用路由本身的规则也是可以实现，具体示例配置如下：

```
flow:
  - name: default_flow
    filter:
      - echo:
          message: "hello world"
  - name: mall_flow
    filter:
      - echo:
          message: "hello mall indices"
  - name: apm_flow
    filter:
      - echo:
          message: "hello apm indices"
  - name: erp_flow
    filter:
      - echo:
          message: "hello erp indices"
router:
  - name: my_router
    default_flow: default_flow
    rules:
      - method:
          - "*"
        pattern:
          - "/apm-{suffix:.*}/"
          - "/apm-{suffix:.*}/{any:.*}"
        flow:
          - apm_flow
      - method:
          - "*"
        pattern:
          - "/erp-{suffix:.*}/"
          - "/erp-{suffix:.*}/{any:.*}"
        flow:
          - erp_flow
      - method:
          - "*"
        pattern:
          - "/mall-{suffix:.*}/"
          - "/mall-{suffix:.*}/{any:.*}"
        flow:
          - mall_flow

```

极限网关功能强大，实现一个功能的方式可以有很多种，这里暂不展开。

## 修改 Kibana 配置

修改 Kibana 的配置文件: `kibana.yml`，替换 Elasticsearch 的地址为网关地址(`http://192.168.3.200:8000`)，如下：

```
elasticsearch.hosts: ["http://192.168.3.200:8000"]
```

重启 Kibana 让配置生效。

## 效果如下

{{% load-img "/img/kibana-clusters-dev.jpg" "" %}}

可以看到，在一个 Kibana 的开发者工具里面，我们已经可以像操作一个集群一样来同时读写实际上来自三个不同集群的索引数据了。

## 展望

通过极限网关，我们还可以非常灵活的进行在线请求的流量编辑，动态组合不同集群的操作。
