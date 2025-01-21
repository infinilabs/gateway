---
title: "flow"
---

# flow

## 描述

flow 过滤器用来跳转或执行某个或一系列其他流程。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: flow
    filter:
      - flow:
          flows:
          - request_logging
```

使用上下文的动态 Flow:

```
flow:
  - name: dns-flow
    filter:
      - flow:
          ignore_undefined_flow: true
          context_flow:
            context: _ctx.request.host
            context_parse_pattern: (?P<uuid>^[0-9a-z_\-]+)\.
            flow_id_template: flow_$[[uuid]]
      - set_response:
          status: 503
          content_type: application/json
          body: '{"message":"invalid HOST"}'

```

支持的上下文变量，请访问 [上下文](../context/) .

## 参数说明

| 名称                               | 类型   | 说明                                                             |
| ---------------------------------- | ------ | ---------------------------------------------------------------- |
| flow                               | string | 流程 ID，支持指定单个 flow 执行                                  |
| flows                              | array  | 流程 ID，数组格式，可以指定多个，依次执行                        |
| ignore_undefined_flow              | bool   | 是否忽略未知的 flow，继续执行                                    |
| context_flow.context               | string | 用来查找 flow_id 的上下文变量                                    |
| context_flow.context_parse_pattern | string | 用来抽取变量的正则表达式                                         |
| context_flow.flow_id_template      | string | 用来生成 flow_id 的模版                                          |
| context_flow.continue              | string | 上下文映射的 Flow 执行完毕之后是否继续下一个过滤器，默认 `false` |
