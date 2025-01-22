---
title: "context_limiter"
---

# context_limiter

## 描述

context_limiter 过滤器用来按照请求上下文来进行限速。

## 配置示例

配置示例如下：

```
flow:
  - name: default_flow
    filter:
      - context_limiter:
          max_requests: 1
          action: drop
          context:
            - _ctx.request.path
            - _ctx.request.header.Host
            - _ctx.request.header.Env
```

上面的配置中，对 `_ctx.request.path` 、 `_ctx.request.header.Host` 和 `_ctx.request.header.Env` 这三个上下文变量来组成一个 bucket 进行限速。
允许的最大 qps 为 `1`每秒，达到限速直接拒绝范围外的后续请求。

## 参数说明

| 名称                 | 类型   | 说明                                                                |
| -------------------- | ------ | ------------------------------------------------------------------- |
| context              | array  | 设置上下文变量，依次组合成一个 bucket key                           |
| interval             | string | 评估限速的单位时间间隔，默认为 `1s`                                 |
| max_requests         | int    | 单位间隔内最大的请求次数限额                                        |
| burst_requests       | int    | 单位间隔内极限允许的请求次数                                        |
| max_bytes            | int    | 单位间隔内最大的请求流量限额                                        |
| burst_bytes          | int    | 单位间隔内极限允许的流量限额                                        |
| action               | string | 触发限速之后的处理动作，分为 `retry` 和 `drop` 两种，默认为 `retry` |
| status               | string | 设置达到限速条件的返回状态码，默认 `429`                            |
| message              | string | 设置达到限速条件的请求的拒绝返回消息                                |
| retry_delay_in_ms    | int    | 限速重试的时间间隔，单位毫秒，默认 `10`，即 10 毫秒                 |
| max_retry_times      | int    | 限速重试的最大重试次数，默认 `1000`                                 |
| failed_retry_message | string | 设置达到最大重试次数的请求的拒绝返回消息                            |
| log_warn_message     | bool   | 是否输出警告消息到日志                                              |
