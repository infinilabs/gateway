---
title: "drop"
---

# drop

## 描述

drop 过滤器用来丢弃某个消息，提前结束请求的处理。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: drop
    filter:
      - drop:
```
