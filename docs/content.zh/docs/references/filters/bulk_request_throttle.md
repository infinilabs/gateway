---
title: "bulk_request_throttle"
---

# bulk_request_throttle

## 描述

bulk_request_throttle 过滤器用来对 Elasticsearch 的 Bulk 请求进行限速。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: bulk_request_mutate
    filter:
      - bulk_request_throttle:
          indices:
            test:
              max_requests: 5
              action: drop
              message: "test writing too fast。"
              log_warn_message: true
            filebeat-*:
              max_bytes: 512
              action: drop
              message: "filebeat indices writing too fast。"
              log_warn_message: true
```

## 参数说明

| 名称                                | 类型   | 说明                                                                |
| ----------------------------------- | ------ | ------------------------------------------------------------------- |
| indices                             | map    | 用于限速的索引，可以分别设置限速规则                                |
| indices.[NAME].interval             | string | 评估限速的单位时间间隔，默认为 `1s`                                 |
| indices.[NAME].max_requests         | int    | 单位间隔内最大的请求次数限额                                        |
| indices.[NAME].burst_requests       | int    | 单位间隔内极限允许的请求次数                                        |
| indices.[NAME].max_bytes            | int    | 单位间隔内最大的请求流量限额                                        |
| indices.[NAME].burst_bytes          | int    | 单位间隔内极限允许的流量限额                                        |
| indices.[NAME].action               | string | 触发限速之后的处理动作，分为 `retry` 和 `drop` 两种，默认为 `retry` |
| indices.[NAME].status               | string | 设置达到限速条件的返回状态码，默认 `429`                            |
| indices.[NAME].message              | string | 设置达到限速条件的请求的拒绝返回消息                                |
| indices.[NAME].retry_delay_in_ms    | int    | 限速重试的时间间隔，单位毫秒，默认 `10`，即 10 毫秒                 |
| indices.[NAME].max_retry_times      | int    | 限速重试的最大重试次数，默认 `1000`                                 |
| indices.[NAME].failed_retry_message | string | 设置达到最大重试次数的请求的拒绝返回消息                            |
| indices.[NAME].log_warn_message     | bool   | 是否输出警告消息到日志                                              |
