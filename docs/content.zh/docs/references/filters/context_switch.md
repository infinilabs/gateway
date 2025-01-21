---
title: "context_switch"
---

# context_switch

## 描述

context_switch 过滤器用来使用上下文变量来进行条件判断实现灵活跳转。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: context_switch
    filter:
      - context_switch:
          context: logging.month
          default_flow: echo_message_not_found
          switch:
            - case: ["02","01"]
              action: redirect_flow
              flow: echo_message_01_02
            - case: ["03"]
              action: redirect_flow
              flow: echo_message_03
```

## 参数说明

| 名称             | 类型     | 说明                                                                              |
| ---------------- | -------- | --------------------------------------------------------------------------------- |
| context          | string   | 上下文变量名称                                                                    |
| skip_error       | bool     | 是否忽略错误直接返回，如上下文变量不存在                                          |
| default_action   | string   | 默认的执行动作，支持 `redirect_flow` 和 `drop`，默认 `redirect_flow`              |
| default_flow     | string   | 默认的 flow 名称                                                                  |
| stringify_value  | bool     | 是否将参数都统一成字符来进行处理，默认 `true`。                                   |
| continue         | bool     | 匹配跳转之后，是否还继续执行后面的流程，设置成 `false` 则立即返回，默认 `false`。 |
| switch           | array    | 条件判断枚举数组                                                                  |
| switch[i].case   | []string | 符合匹配条件的字符枚举                                                            |
| switch[i].action | string   | 匹配之后的执行动作，支持 `redirect_flow` 和 `drop`，默认 `redirect_flow`          |
| switch[i].flow   | string   | 如果动作是 `redirect_flow`，则跳转到该 flow，否则执行默认的 flow                  |
