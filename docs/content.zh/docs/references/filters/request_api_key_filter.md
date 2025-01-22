---
title: "request_api_key_filter"
---

# request_api_key_filter

## 描述

当 Elasticsearch 是通过 API Key 方式来进行身份认证的时候，request_api_key_filter 过滤器可用来按请求的 API ID 来进行过滤。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - request_api_key_filter:
          message: "Request filtered!"
          exclude:
            - VuaCfGcBCdbkQm-e5aOx
```

上面的例子表示，来自 `VuaCfGcBCdbkQm-e5aOx` 的请求会被拒绝，如下。

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
< HTTP/1.1 403 Forbidden
< Server: INFINI
< Date: Mon, 12 Apr 2021 15:02:37 GMT
< content-type: text/plain; charset=utf-8
< content-length: 17
< FILTERED: true
< process: request_api_key_filter
<
* Connection #0 to host localhost left intact
{"error":true,"message":"Request filtered!"}%                                                              ➜  ~
```

## 参数说明

| 名称    | 类型   | 说明                                                                        |
| ------- | ------ | --------------------------------------------------------------------------- |
| exclude | array  | 拒绝通过的请求的用户名列表                                                  |
| include | array  | 允许通过的请求的用户名列表                                                  |
| action  | string | 符合过滤条件之后的处理动作，可以是 `deny` 和 `redirect_flow`，默认为 `deny` |
| status  | int    | 自定义模式匹配之后返回的状态码                                              |
| message | string | 自定义 `deny` 模式返回的消息文本                                            |
| flow    | string | 自定义 `redirect_flow` 模式执行的 flow ID                                   |

{{< hint info >}}
注意: 当设置了 `include` 条件的情况下，必须至少满足 `include` 设置的其中一种响应码才能被允许通过。
当仅设置了 `exclude` 条件的情况下，不符合 `exclude` 的任意请求都允许通过。
{{< /hint >}}
