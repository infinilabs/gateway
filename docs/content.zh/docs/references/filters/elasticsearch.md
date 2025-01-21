---
title: "elasticsearch"
---

# elasticsearch

## 描述

elasticsearch 过滤器是一个用于请求转发给后端 Elasticsearch 集群的过滤器。

## 配置示例

使用 elasticsearch 过滤器之前，需要提前定义一个 Elasticsearch 的集群配置节点，如下：

```
elasticsearch:
- name: prod
  enabled: true
  endpoint: http://192.168.3.201:9200
```

流程的配置示例如下：

```
flow:
  - name: cache_first
    filter:
      - elasticsearch:
          elasticsearch: prod
```

上面的例子即将请求转发给 `prod` 集群。

## 自动更新

对于一个大规模的集群，可能存在很多的节点，不可能一一配置后端的所有节点，只需要先指定 Elasticsearch 模块允许自动发现后端节点，如下：

```
elasticsearch:
- name: prod
  enabled: true
  endpoint: http://192.168.3.201:9200
  discovery:
    enabled: true
    refresh:
      enabled: true
  basic_auth:
    username: elastic
    password: pass
```

然后过滤器这边的配置也开启刷新，即可访问后端所有节点，且节点上下线也会自动更新，示例如下：

```
flow:
  - name: cache_first
    filter:
      - elasticsearch:
          elasticsearch: prod
          refresh:
            enabled: true
            interval: 30s
```

## 设置权重

如果后端集群很多，极限网关支持对不同的节点设置不同的访问权重，配置示例如下：

```
flow:
  - name: cache_first
    filter:
      - elasticsearch:
          elasticsearch: prod
          balancer: weight
          refresh:
            enabled: true
            interval: 30s
          weights:
            - host: 192.168.3.201:9200
              weight: 10
            - host: 192.168.3.202:9200
              weight: 20
            - host: 192.168.3.203:9200
              weight: 30
```

上面的例子中，发往 Elasticsearch 集群的流量，将以 `3：2：1` 的比例分别发给 `203`、`202` 和 `201` 这三个节点。

## 过滤节点

极限网关还支持按照节点的 IP、标签、角色来进行过滤，可以用来将请求避免发送给特定的节点，如 Master、冷节点等，配置示例如下：

```
flow:
  - name: cache_first
    filter:
      - elasticsearch:
          elasticsearch: prod
          balancer: weight
          refresh:
            enabled: true
            interval: 30s
          filter:
            hosts:
              exclude:
                - 192.168.3.201:9200
              include:
                - 192.168.3.202:9200
                - 192.168.3.203:9200
            tags:
              exclude:
                - temp: cold
              include:
                - disk: ssd
            roles:
              exclude:
                - master
              include:
                - data
                - ingest
```

## 参数说明

| 名称                     | 类型     | 说明                                                                                                                                                                    |
| ------------------------ | -------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| elasticsearch            | string   | Elasticsearch 集群的名称                                                                                                                                                |
| max_connection_per_node  | int      | 限制访问 Elasticsearch 集群每个节点的最大 TCP 连接数，默认 `5000`                                                                                                       |
| max_response_size        | int      | 限制 Elasticsearch 请求返回的最大消息体大小，默认 `100*1024*1024`                                                                                                       |
| max_retry_times          | int      | 限制 Elasticsearch 出错的重试次数，默认 `0`                                                                                                                             |
| max_conn_wait_timeout    | duration | 限制 Elasticsearch 等待空闲链接的超时时间，默认 `30s`                                                                                                                   |
| max_idle_conn_duration   | duration | 限制 Elasticsearch 连接的空闲时间，默认 `30s`                                                                                                                           |
| max_conn_duration        | duration | 限制 Elasticsearch 连接的持续时间，默认 `0s`                                                                                                                            |
| timeout                  | duration | 等待 Elasticsearch 请求返回超时时间，默认 `30s`。警告：`timeout` 不会终止请求本身。请求将在后台继续，响应将被丢弃。如果请求时间过长并且连接池已满，请尝试设置读取超时。 |
| dial_timeout             | duration | 限制 Elasticsearch 请求的 dial 超时时间，默认`3s`                                                                                                                       |
| read_timeout             | duration | 限制 Elasticsearch 请求的读取超时时间，默认 `0s`                                                                                                                        |
| write_timeout            | duration | 限制 Elasticsearch 请求的写入超时时间，默认 `0s`                                                                                                                        |
| read_buffer_size         | int      | 设置 Elasticsearch 请求的读缓存大小，默认 `4096*4`                                                                                                                      |
| write_buffer_size        | int      | 设置 Elasticsearch 请求的写缓存大小，默认 `4096*4`                                                                                                                      |
| tls_insecure_skip_verify | bool     | 是否忽略 Elasticsearch 集群的 TLS 证书校验，默认 `true`                                                                                                                 |
| balancer                 | string   | 后端 Elasticsearch 节点的负载均衡算法，目前只有 `weight` 基于权重的算法                                                                                                 |
| skip_metadata_enrich     | bool   | 是否跳过 Elasticsearch 元数据的处理，不添加 `X-*` 元数据到请求和响应的头信息                      |
| refresh.enable           | bool     | 是否开启节点状态变化的自动刷新，可感知后端 Elasticsearch 拓扑的变化                                                                                                     |
| refresh.interval         | int      | 节点状态刷新的间隔时间                                                                                                                                                  |
| weights                  | array    | 可以设置后端节点的优先级，权重高的转发请求的比例相应提高                                                                                                                |
| filter                   | object   | 后端 Elasticsearch 节点的过滤规则，可以将请求转发给特定的节点                                                                                                           |
| filter.hosts             | object   | 按照 Elasticsearch 的访问地址来进行过滤                                                                                                                                 |
| filter.tags              | object   | 按照 Elasticsearch 的标签来进行过滤                                                                                                                                     |
| filter.roles             | object   | 按照 Elasticsearch 的角色来进行过滤                                                                                                                                     |
| filter.\*.exclude        | array    | 排除特定的条件，任何匹配的节点会被拒绝执行请求的代理                                                                                                                    |
| filter.\*.include        | array    | 允许符合条件的 Elasticsearch 节点来代理请求，在 exclude 参数没有配置的情况下，如果配置了 include 条件，则必须要满足任意一个 include 条件，否则不允许进行请求的代理      |
