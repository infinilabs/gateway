---
title: "request_method_filter"
asciinema: true
---

# request_method_filter

## 描述

request_method_filter 过滤器用来按请求 Method 来过滤流量。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - request_method_filter:
          exclude:
            - PUT
            - POST
          include:
            - GET
            - HEAD
            - DELETE
```

## 参数说明

| 名称    | 类型   | 说明                                                                        |
| ------- | ------ | --------------------------------------------------------------------------- |
| exclude | array  | 拒绝通过的请求 Method                                                       |
| include | array  | 允许通过的请求 Method                                                       |
| action  | string | 符合过滤条件之后的处理动作，可以是 `deny` 和 `redirect_flow`，默认为 `deny` |
| status  | int    | 自定义模式匹配之后返回的状态码                                              |
| message | string | 自定义 `deny` 模式返回的消息文本                                            |
| flow    | string | 自定义 `redirect_flow` 模式执行的 flow ID                                   |

{{< hint info >}}
注意: 当设置了 `include` 条件的情况下，必须至少满足 `include` 设置的其中一种响应码才能被允许通过。
当仅设置了 `exclude` 条件的情况下，不符合 `exclude` 的任意请求都允许通过。
{{< /hint >}}
