---
title: "服务入口"
weight: 20
---

# 服务入口

## 定义入口

每一个网关都至少要对外暴露一个服务的入口，用来接收业务的操作请求，这个在极限网关里面叫做 `entry`，通过下面的参数即可定义：

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

通过参数 `network.binding` 可以指定服务监听的 IP 和地址，极限网关支持端口重用，也就是多个极限网关共享一个相同的 IP 和端口，这样可以充分利用服务器的资源，
也能做到不同网关进程的动态配置修改（通过开启多个进程，修改配置之后，依次重启各个进程）而不会中断客户端的正常请求。

每个发送到 `entry` 的请求都会通过 `router` 来进行流量的路由处理，`router` 在单独的地方定义规则，以方便在不同的 `entry` 间复用，`entry` 只需要通过 `router` 参数指定要使用的 `router` 规则即可，这里定义的是 `default`。

## TLS 配置

极限网关支持无缝开启 TLS 传输加密，只需要将 `tls.enabled` 设置成 `true`，即可直接切换为 HTTPS 的通信模式，极限网关能自动生成自签证书。

极限网关也支持自定义证书路径，配置方式如下：

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

## 多个服务

极限网关支持一个网关监听多个不同的服务入口，各个服务入口的监听地址、协议和路由都可以分别定义，用来满足不同的业务需求，配置示例如下：

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

上面的例子，定义了一个名为 `es_ingest` 的服务入口，监听的地址是 `0.0.0.0:8000`，所有请求都通过 `ingest_router` 来进行处理。
另外一个 `es_search` 服务，监听端口是 `9000`，使用 `search_router` 来进行请求处理，可以实现业务的读写分离。
另外，对于不同的后端 Elasticsearch 集群也可以定义不同的服务入口，通过网关来进行请求的代理转发。

## IPv6 支持

极限网关支持绑定到 IPv6 地址，示例如下：

```
entry:
  - name: es_ingest
    enabled: true
    router: ingest_router
    network:
#      binding: "[ff80::4e2:7fb6:7db6:a839%en0]:8000"
      binding: "[::]:8000"
```

## 参数说明

| 名称                       | 类型   | 说明                                            |
| -------------------------- | ------ | ----------------------------------------------- |
| name                       | string | 服务入口名称                                    |
| enabled                    | bool   | 是否启用该入口                                  |
| max_concurrency            | int    | 最大的并发连接数，默认 `10000`                  |
| router                     | string | 路由名称                                        |
| network                    | object | 网络的相关配置                                  |
| tls                        | object | TLS 安全传输相关配置                            |
| network.host               | string | 服务监听的网络地址，如：`192.168.3.10`          |
| network.port               | int    | 服务监听的端口地址，如：`8000`                  |
| network.binding            | string | 服务监听的网络绑定地址，如：`0.0.0.0:8000`      |
| network.publish            | string | 服务监听的对外访问地址，如：`192.168.3.10:8000` |
| network.reuse_port         | bool   | 是否重用网络端口，用于多进程端口共享            |
| network.skip_occupied_port | bool   | 是否自动跳过已占用端口                          |
| tls.enabled                | bool   | 是否启用 TLS 安全传输                           |
| tls.cert_file              | string | TLS 安全证书公钥路径                            |
| tls.key_file               | string | TLS 安全证书秘钥路径                            |
| tls.skip_insecure_verify   | bool   | 是否忽略 TLS 的证书校验                         |
