---
title: "retry_limiter"
---

# retry_limiter

## 描述

retry_limiter 过滤器用来判断一个请求是否达到最大重试次数，避免一个请求的无限重试。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: retry_limiter
    filter:
      - retry_limiter:
          queue_name: "deadlock_messages"
          max_retry_times: 3
```

## 参数说明

| 名称            | 类型   | 说明                                             |
| --------------- | ------ | ------------------------------------------------ |
| max_retry_times | int    | 最大重试次数，默认为 `3`                         |
| queue_name      | string | 达到重试最大次数后，输出消息到指定消息队列的名称 |
| tag_on_success  | array  | 触发重试条件之后，请求上下文打上指定标记         |
