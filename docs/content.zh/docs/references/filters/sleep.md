---
title: "sleep"
---

# sleep

## 描述

sleep 过滤器用来添加一个固定的延迟到请求，可以人为降速。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: slow_query_logging_test
    filter:
      - sleep:
          sleep_in_million_seconds: 1024
```

## 参数说明

| 名称                     | 类型  | 说明                           |
| ------------------------ | ----- | ------------------------------ |
| sleep_in_million_seconds | int64 | 需要添加的延迟长度，单位为毫秒 |
