---
title: "Elasticsearch"
weight: 47
---

# Elasticsearch

## 定义资源

极限网关支持多集群的访问，支持不同的版本，每个集群作为一个 Elasticsearch 后端资源，可以后续被极限网关的多个地方使用，以下面的这个例子为例：

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

上面的例子定义了一个名为 `local` 的本地开发测试集群，和一个名为 `dev` 的开发集群。开发集群开启了身份验证，这里也定义了相应的用户名和密码。
最后还定义了一个名为 `prod` 的生产集群，并且通过参数 `discovery` 开启了集群的节点拓扑自动发现和更新。

## 参数说明

| 名称                                    | 类型   | 说明                                                                                          |
| --------------------------------------- | ------ | --------------------------------------------------------------------------------------------- |
| name                                    | string | Elasticsearch 集群名称                                                                        |
| project                                 | string | 项目名称                                                                                      |
| location.provider                       | string | 集群提供商                                                                                    |
| location.region                         | string | 集群所在可用区                                                                                |
| location.dc                             | string | 集群所在数据中心                                                                              |
| location.rack                           | string | 集群所在机架                                                                                  |
| labels                                  | map    | 集群自定义标签                                                                                |
| tags                                    | array  | 集群自定义标签                                                                                |
| enabled                                 | bool   | 是否启用                                                                                      |
| endpoint                                | string | Elasticsearch 访问地址，如: `http://localhost:9200`                                           |
| endpoints                               | array  | Elasticsearch 访问地址列表，支持多个入口地址，用于冗余                                        |
| schema                                  | string | 协议类型，`http` 或者 `https`                                                                 |
| host                                    | string | Elasticsearch 主机，格式：`localhost:9200`，host 和 endpoint 任意选择一种配置方式即可         |
| hosts                                   | array  | Elasticsearch 主机列表，支持多个入口地址，用于冗余                                            |
| request_timeout                         | int    | 请求超时时间，单位秒，默认 `30`                                                               |
| request_compress                        | bool   | 是否开启 Gzip 压缩                                                                            |
| basic_auth                              | object | 身份认证信息                                                                                  |
| basic_auth.username                     | string | 用户名                                                                                        |
| basic_auth.password                     | string | 密码                                                                                          |
| discovery                               | object | 集群发现设置                                                                                  |
| discovery.enabled                       | bool   | 是否启用集群拓扑发现                                                                          |
| discovery.refresh                       | object | 集群拓扑更新设置                                                                              |
| discovery.refresh.enabled               | bool   | 是否启用集群拓扑自动更新                                                                      |
| discovery.refresh.interval              | string | 集群拓扑自动更新时间间隔                                                                      |
| traffic_control                         | object | 集群按节点级别的总体流量控制                                                                  |
| traffic_control.enabled      | bool    | 是否启用限速                                                                      |
| traffic_control.max_bytes_per_node      | int    | 最大允许的每秒请求字节数                                                                      |
| traffic_control.max_qps_per_node        | int    | 最大允许的每秒请求次数，不区分读写                                                            |
| traffic_control.max_connection_per_node | int    | 最大允许的主机连接数                                                                          |
| traffic_control.max_wait_time_in_ms     | int    | 如遇限速, 最大允许的等待时间,默认 `10000`                                                     |
| allow_access_when_master_not_found      | bool   | 当集群出现 `master_not_discovered_exception` 异常后，任然允许转发请求到该集群，默认为 `false` |
