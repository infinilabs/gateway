---
title: "response_header_format"
---

# response_header_format

## 描述

response_header_format 过滤器用来将请求响应的 Header 信息里面的 Key 都转换成小写。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - response_header_format:
```
