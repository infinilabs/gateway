---
title: "set_context"
---

# set_context

## 描述

set_context 过滤器用来设置请求上下文的相关信息。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - set_response:
          body: '{"message":"hello world"}'
      - set_context:
          context:
#            _ctx.request.uri: http://baidu.com
#            _ctx.request.path: new_request_path
#            _ctx.request.host: api.infinilabs.com
#            _ctx.request.method: DELETE
#            _ctx.request.body: "hello world"
#            _ctx.request.body_json.explain: true
#            _ctx.request.query_args.from: 100
#            _ctx.request.header.ENV: dev
#            _ctx.response.content_type: "application/json"
#            _ctx.response.header.TIMES: 100
#            _ctx.response.status: 419
#            _ctx.response.body: "new_body"
            _ctx.response.body_json.success: true
      - dump:
          request: true
```

## 参数说明

| 名称    | 类型 | 说明                     |
| ------- | ---- | ------------------------ |
| context | map  | 请求的上下文及对应的新值 |

支持的上下文变量列表如下：

| 名称                                 | 类型   | 说明                 |
| ------------------------------------ | ------ | -------------------- |
| \_ctx.request.uri                    | string | 完整请求的 URL 地址  |
| \_ctx.request.path                   | string | 请求的路径           |
| \_ctx.request.host                   | string | 请求的主机           |
| \_ctx.request.method                 | string | 请求 Method 类型     |
| \_ctx.request.body                   | string | 请求体               |
| \_ctx.request.body_json.[JSON_PATH]  | string | JSON 请求对象的 Path |
| \_ctx.request.query_args.[KEY]       | string | URL 查询请求参数     |
| \_ctx.request.header.[KEY]           | string | 请求头信息           |
| \_ctx.response.content_type          | string | 请求体类型           |
| \_ctx.response.header.[KEY]          | string | 返回头信息           |
| \_ctx.response.status                | int    | 返回状态码           |
| \_ctx.response.body                  | string | 返回响应体           |
| \_ctx.response.body_json.[JSON_PATH] | string | JSON 返回对象的 Path |
