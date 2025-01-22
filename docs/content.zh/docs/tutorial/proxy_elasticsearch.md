---
title: "为 Elasticsearch 无缝添加代理和基础安全"
weight: 100
---

# 为 Elasticsearch 无缝添加代理和基础安全

如果你的 Elasticsearch 版本比较多或者比较旧，或者没有设置 TLS 和身份信息，那么任何人都有可能直接访问 Elasticsearch，而使用极限网关可以快速的进行修复。

## 使用 Elasticsearch 过滤器来转发请求

首先定义一个 Elasticsearch 的资源，如下：

```
elasticsearch:
- name: prod
  enabled: true
  endpoint: http://192.168.3.201:9200
```

然后可以使用如下的过滤器来转发请求到上面定义的 Elasticsearch 资源，名称为 `prod`：

```
  - elasticsearch:
      elasticsearch: prod
```

有关该过滤器的更多详情，请参考文档：[elasticsearch filter](../references/filters/elasticsearch/)

## 添加一个简单的身份验证

我们进行添加一个基础的身份验证，来限制目标集群的访问

```
  - basic_auth:
      valid_users:
        medcl: passwd
```

## 开启 TLS

如果设置了身份，但是没有设置 TLS 也是不行的，因为 HTTP 是明文传输协议，可以非常容易泄露密码，配置如下：

```
  - name: my_es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8000
    tls:
      enabled: true
```

通过地址 `https://localhost:8000` 就可以访问到 `prod` 的 Elasticsearch 集群。

注意的是，这里监听的地址是 `0.0.0.0`，代表本机所有网卡上的 IP 都进行了监听，
为了安全起见，你可能需要修改为只监听本地地址或者指定的网卡 IP 地址。

## 兼容 HTTP 访问

如果存在遗留的系统没有办法切换到新集群的，可以提供一个新的端口来进行 HTTP 的访问：

```
  - name: my_unsecure_es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8001
    tls:
      enabled: false
```

通过地址 `http://localhost:8001` 就可以访问到 `prod` 的 Elasticsearch 集群。

## 完整配置如下

```
elasticsearch:
- name: prod
  enabled: true
  endpoint: http://192.168.3.201:9200

entry:
  - name: my_es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8000
    tls:
      enabled: true
  - name: my_unsecure_es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8001
    tls:
      enabled: false

flow:
  - name: default_flow
    filter:
      - basic_auth:
          valid_users:
            medcl: passwd
      - elasticsearch:
          elasticsearch: prod
router:
  - name: my_router
    default_flow: default_flow
```

## 效果如下

现在使用网关来访问 Elasticsearch 就需要登陆了，如下：

{{% load-img "/img/elasticsearch-login.jpg" "" %}}
