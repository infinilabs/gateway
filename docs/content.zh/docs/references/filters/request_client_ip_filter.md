---
title: "request_client_ip_filter"
---

# request_client_ip_filter

## 描述

request_client_ip_filter 过滤器用来按请求的来源用户 IP 信息来过滤流量。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - request_client_ip_filter:
          exclude:
          - 192.168.3.67
```

上面的例子表示，来自 `192.168.3.67` 的请求不允许通过。

路由跳转的例子:

```
flow:
  - name: echo
    filter:
      - echo:
          message: hello stanger
  - name: default_flow
    filter:
      - request_client_ip_filter:
          action: redirect_flow
          flow: echo
          exclude:
            - 192.168.3.67
```

来自 `192.168.3.67` 会跳转到另外的 `echo` 流程。

## 参数说明

| 名称    | 类型   | 说明                                                                        |
| ------- | ------ | --------------------------------------------------------------------------- |
| exclude | array  | 拒绝通过的请求 IP 数组列表                                                  |
| include | array  | 允许通过的请求 IP 数组列表                                                  |
| action  | string | 符合过滤条件之后的处理动作，可以是 `deny` 和 `redirect_flow`，默认为 `deny` |
| status  | int    | 自定义模式匹配之后返回的状态码                                              |
| message | string | 自定义 `deny` 模式返回的消息文本                                            |
| flow    | string | 自定义 `redirect_flow` 模式执行的 flow ID                                   |

{{< hint info >}}
注意: 当设置了 `include` 条件的情况下，必须至少满足 `include` 设置的其中一种响应码才能被允许通过。
当仅设置了 `exclude` 条件的情况下，不符合 `exclude` 的任意请求都允许通过。
{{< /hint >}}
