---
title: "request_api_key_limiter"
asciinema: true
---

# request_api_key_limiter

## 描述

request_api_key_limiter 过滤器用来按照 API Key 来进行限速。

## 配置示例

配置示例如下：

```
flow:
  - name: rate_limit_flow
    filter:
     - request_api_key_limiter:
         id:
           - VuaCfGcBCdbkQm-e5aOx
         max_requests: 1
         action: drop # retry or drop
         message: "your api_key reached our limit"
```

上面的配置中，对 `VuaCfGcBCdbkQm-e5aOx` 这个 API ID 进行限速，允许的最大 qps 为 `1` 每秒。

```
➜  ~ curl localhost:8000 -H "Authorization: ApiKey VnVhQ2ZHY0JDZGJrUW0tZTVhT3g6dWkybHAyYXhUTm1zeWFrdzl0dk5udw==" -v
* Rebuilt URL to: localhost:8000/
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 8000 (#0)
> GET / HTTP/1.1
> Host: localhost:8000
> User-Agent: curl/7.54.0
> Accept: */*
> Authorization: ApiKey VnVhQ2ZHY0JDZGJrUW0tZTVhT3g6dWkybHAyYXhUTm1zeWFrdzl0dk5udw==
>
< HTTP/1.1 429 Too Many Requests
< Server: INFINI
< Date: Mon, 12 Apr 2021 15:14:52 GMT
< content-type: text/plain; charset=utf-8
< content-length: 30
< process: request_api_key_limiter
<
* Connection #0 to host localhost left intact
your api_key reached our limit%
```

## 参数说明

| 名称                 | 类型   | 说明                                                                |
| -------------------- | ------ | ------------------------------------------------------------------- |
| id                   | array  | 设置哪些 API ID 会参与限速，不设置表示所有 API Key 参与             |
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
