---
title: "request_host_limiter"
asciinema: true
---

# request_host_limiter

## 描述

request_host_limiter 过滤器用来按照请求主机（域名）来进行限速。

## 配置示例

配置示例如下：

```
flow:
  - name: rate_limit_flow
    filter:
      - request_host_limiter:
          host:
            - api.elasticsearch.cn:8000
            - logging.elasticsearch.cn:8000
          max_requests: 256
#          max_bytes: 102400 #100k
          action: retry # retry or drop
#          max_retry_times: 1000
#          retry_interval: 500 #100ms
          message: "you reached our limit"
```

上面的配置中，对 `api.elasticsearch.cn` 和 `logging.elasticsearch.cn` 这两个访问域名进行限速，允许的最大 qps 为 `256` 每秒。

## 参数说明

| 名称                 | 类型   | 说明                                                                                                                |
| -------------------- | ------ | ------------------------------------------------------------------------------------------------------------------- |
| host                 | array  | 设置哪些主机域名会参与限速，不设置表示都参与，注意，如果访问的域名带端口号，这里也需包含端口号，如 `localhost:8080` |
| interval             | string | 评估限速的单位时间间隔，默认为 `1s`                                                                                 |
| max_requests         | int    | 单位间隔内最大的请求次数限额                                                                                        |
| burst_requests       | int    | 单位间隔内极限允许的请求次数                                                                                        |
| max_bytes            | int    | 单位间隔内最大的请求流量限额                                                                                        |
| burst_bytes          | int    | 单位间隔内极限允许的流量限额                                                                                        |
| action               | string | 触发限速之后的处理动作，分为 `retry` 和 `drop` 两种，默认为 `retry`                                                 |
| status               | string | 设置达到限速条件的返回状态码，默认 `429`                                                                            |
| message              | string | 设置达到限速条件的请求的拒绝返回消息                                                                                |
| retry_delay_in_ms    | int    | 限速重试的时间间隔，单位毫秒，默认 `10`，即 10 毫秒                                                                 |
| max_retry_times      | int    | 限速重试的最大重试次数，默认 `1000`                                                                                 |
| failed_retry_message | string | 设置达到最大重试次数的请求的拒绝返回消息                                                                            |
| log_warn_message     | bool   | 是否输出警告消息到日志                                                                                              |
