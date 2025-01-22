---
title: "context_regex_replace"
---

# context_regex_replace

## 描述

context_regex_replace 过滤器用来通过正则表达式来替换修改请求上下文的相关信息。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - context_regex_replace:
          context: "_ctx.request.path"
          pattern: "^/"
          to: "/cluster:"
          when:
            contains:
              _ctx.request.path: /_search
      - dump:
          request: true
```

这个例子可以将请求 `curl localhost:8000/abc/_search` 替换为 `curl localhost:8000/cluster:abc/_search`

## 参数说明

| 名称    | 类型   | 说明                     |
| ------- | ------ | ------------------------ |
| context | string | 请求的上下文及对应的 Key |
| pattern | string | 用于匹配替换的正则表达式 |
| to      | string | 替换为目标的字符串内容   |

支持修改的上下文变量列表如下：

| 名称                                 | 类型   | 说明                 |
| ------------------------------------ | ------ | -------------------- |
| \_ctx.request.uri                    | string | 完整请求的 URL 地址  |
| \_ctx.request.path                   | string | 请求的路径           |
| \_ctx.request.host                   | string | 请求的主机           |
| \_ctx.request.body                   | string | 请求体               |
| \_ctx.request.body_json.[JSON_PATH]  | string | JSON 请求对象的 Path |
| \_ctx.request.query_args.[KEY]       | string | URL 查询请求参数     |
| \_ctx.request.header.[KEY]           | string | 请求头信息           |
| \_ctx.response.header.[KEY]          | string | 返回头信息           |
| \_ctx.response.body                  | string | 返回响应体           |
| \_ctx.response.body_json.[JSON_PATH] | string | JSON 返回对象的 Path |
