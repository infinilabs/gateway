---
title: "http"
---

# http

## 描述

http 过滤器用来将请求代理转发到指定的 http 服务器。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: default_flow
    filter:
      - basic_auth:
          valid_users:
            medcl: passwd
      - http:
          schema: "http" #https or http
          #host: "192.168.3.98:5601"
          hosts:
           - "192.168.3.98:5601"
           - "192.168.3.98:5602"
```

## 参数说明

| 名称                     | 类型     | 说明                                      |
| ------------------------ | -------- | ----------------------------------------- |
| schema                   | string   | `http` 或是 `https`                       |
| host                     | string   | 目标主机地址，带端口，如 `localhost:9200` |
| hosts                    | array    | 主机地址列表，遇到故障，依次尝试          |
| skip_failure_host        | bool     | 是否跳过不可以的主机，默认 `true`         |
| max_connection_per_node  | int      | 主机的最大连接数，默认 `5000`             |
| max_response_size        | int      | 支持的最大响应体大小                      |
| max_retry_times          | int      | 出错的最大重试次数，默认 `0`              |
| retry_delay_in_ms        | int      | 重试的延迟，默认 `1000`                   |
| skip_cleanup_hop_headers | bool     | 是否移除不兼容的 Hop-by-hop 头信息        |
| max_conn_wait_timeout    | duration | 建立连接的超时时间，默认 `30s`            |
| max_idle_conn_duration   | duration | 空闲连接的超时时间，默认 `30s`            |
| max_conn_duration        | duration | 长连接的超时时间，默认 `0s`               |
| timeout                  | duration | 请求的超时时间，默认 `30s`                |
| read_timeout             | duration | 读请求的超时时间，默认 `0s`               |
| write_timeout            | duration | 写请求的超时时间，默认 `0s`               |
| read_buffer_size         | int      | 读请求的缓冲区大小，默认 `16384`          |
| write_buffer_size        | int      | 写请求的缓冲区大小，默认 `16384`          |
| tls_insecure_skip_verify | bool     | 是否忽略 TLS 的校验，默认 `true`          |
