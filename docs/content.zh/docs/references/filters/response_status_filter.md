---
title: "response_status_filter"
asciinema: true
---

# response_status_filter

## 描述

response_status_filter 过滤器用来按后端服务响应的状态码来进行过滤。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - response_status_filter:
          message: "Request filtered!"
          exclude:
            - 404
          include:
            - 200
            - 201
            - 500
```

## 参数说明

| 名称    | 类型   | 说明                                                                        |
| ------- | ------ | --------------------------------------------------------------------------- |
| exclude | array  | 拒绝通过的响应码                                                            |
| include | array  | 允许通过的响应码                                                            |
| action  | string | 符合过滤条件之后的处理动作，可以是 `deny` 和 `redirect_flow`，默认为 `deny` |
| status  | int    | 自定义模式匹配之后返回的状态码                                              |
| message | string | 自定义 `deny` 模式返回的消息文本                                            |
| flow    | string | 自定义 `redirect_flow` 模式执行的 flow ID                                   |

{{< hint info >}}
注意: 当设置了 `include` 条件的情况下，必须至少满足 `include` 设置的其中一种响应码才能被允许通过。
当仅设置了 `exclude` 条件的情况下，不符合 `exclude` 的任意请求都允许通过。
{{< /hint >}}
