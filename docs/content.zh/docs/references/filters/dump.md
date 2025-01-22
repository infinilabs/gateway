---
title: "dump"
asciinema: true
---

# dump

## 描述

dump 过滤器是一个用于在终端打印 Dump 输出相关请求信息的过滤器，主要用于调试。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: hello_world
    filter:
      - dump:
         request: true
         response: true
```

### 参数说明

dump 过滤器比较简单，在需要的流程处理阶段插入 dump 过滤器，即可在终端输出相应阶段的请求信息，方便调试。

| 名称            | 类型  | 说明                       |
| --------------- | ----- | -------------------------- |
| request         | bool  | 是否输出全部完整的请求信息 |
| response        | bool  | 是否输出全部完整的返回信息 |
| uri             | bool  | 是否输出请求的 URI 信息    |
| query_args      | bool  | 是否输出请求的参数信息     |
| user            | bool  | 是否输出请求的用户信息     |
| api_key         | bool  | 是否输出请求的 APIKey 信息 |
| request_header  | bool  | 是否输出请求的头信息       |
| response_header | bool  | 是否输出响应的头信息       |
| status_code     | bool  | 是否输出响应的状态码       |
| context         | array | 输出自定义的上下文信息     |

### 输出上下文

可以使用 `context` 参数来调试请求上下文信息，示例配置文件：

```
flow:
  - name: echo
    filter:
      - set_response:
          status: 201
          content_type: "text/plain; charset=utf-8"
          body: "hello world"
      - set_response_header:
          headers:
            - Env -> Dev
      - dump:
          context:
            - _ctx.id
            - _ctx.tls
            - _ctx.remote_addr
            - _ctx.local_addr
            - _ctx.request.host
            - _ctx.request.method
            - _ctx.request.uri
            - _ctx.request.path
            - _ctx.request.body
            - _ctx.request.body_length
            - _ctx.request.query_args.from
            - _ctx.request.query_args.size
            - _ctx.request.header.Accept
            - _ctx.request.user
            - _ctx.response.status
            - _ctx.response.body
            - _ctx.response.content_type
            - _ctx.response.body_length
            - _ctx.response.header.Env
```

启动网关，执行如下命令：

```
curl http://localhost:8000/medcl/_search\?from\=1\&size\=100 -d'{search:query123}' -v -u 'medcl:123'
```

网关终端输出如下信息：

```
---- dumping context ----
_ctx.id  :  21474836481
_ctx.tls  :  false
_ctx.remote_addr  :  127.0.0.1:50925
_ctx.local_addr  :  127.0.0.1:8000
_ctx.request.host  :  localhost:8000
_ctx.request.method  :  POST
_ctx.request.uri  :  http://localhost:8000/medcl/_search?from=1&size=100
_ctx.request.path  :  /medcl/_search
_ctx.request.body  :  {search:query123}
_ctx.request.body_length  :  17
_ctx.request.query_args.from  :  1
_ctx.request.query_args.size  :  100
_ctx.request.header.Accept  :  */*
_ctx.request.user  :  medcl
_ctx.response.status  :  201
_ctx.response.body  :  hello world
_ctx.response.content_type  :  text/plain; charset=utf-8
_ctx.response.body_length  :  11
_ctx.response.header.Env  :  Dev
```
