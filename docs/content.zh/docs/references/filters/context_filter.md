---
title: "context_filter"
---

# context_filter

## 描述

context_filter 过滤器用来按请求上下文来过滤流量。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - context_filter:
          context: _ctx.request.path
          message: "request not allowed."
          status: 403
          must: #must match all rules to continue
            prefix:
              - /medcl
            contain:
              - _search
            suffix:
              - _search
            wildcard:
              - /*/_search
            regex:
              - ^/m[\w]+dcl
          must_not: # any match will be filtered
            prefix:
              - /.kibana
              - /_security
              - /_security
              - /gateway_requests*
              - /.reporting
              - /_monitoring/bulk
            contain:
              - _refresh
            suffix:
              - _count
              - _refresh
            wildcard:
              - /*/_refresh
            regex:
              - ^/\.m[\w]+dcl
          should:
            prefix:
              - /medcl
            contain:
              - _search
              - _async_search
            suffix:
              - _refresh
            wildcard:
              - /*/_refresh
            regex:
              - ^/m[\w]+dcl
```

## 参数说明

| 名称        | 类型   | 说明                                                                        |
| ----------- | ------ | --------------------------------------------------------------------------- |
| context     | string | 上下文变量                                                                  |
| exclude     | array  | 拒绝通过的请求的变量列表                                                    |
| include     | array  | 允许通过的请求的变量列表                                                    |
| must.\*     | object | 必须都满足所设置条件的情况下才能允许通过                                    |
| must_not.\* | object | 必须都不满足所设置条件的情况下才能通过                                      |
| should.\*   | object | 满足任意所设置条件的情况下即可通过                                          |
| \*.prefix   | array  | 判断是否由特定字符开头                                                      |
| \*.suffix   | array  | 判断是否由特定字符结尾                                                      |
| \*.contain  | array  | 判断是否包含特定字符                                                        |
| \*.wildcard | array  | 判断是否符合通配符匹配规则                                                  |
| \*.regex    | array  | 判断是否符合正则表达式匹配规则                                              |
| action      | string | 符合过滤条件之后的处理动作，可以是 `deny` 和 `redirect_flow`，默认为 `deny` |
| status      | int    | 自定义模式匹配之后返回的状态码                                              |
| message     | string | 自定义 `deny` 模式返回的消息文本                                            |
| flow        | string | 自定义 `redirect_flow` 模式执行的 flow ID                                   |

Note: 当仅设置了 `should` 条件的情况下，必须至少满足 `should` 设置的其中一种才能被允许通过。
