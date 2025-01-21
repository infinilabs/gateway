---
title: "set_response_header"
---

# set_response_header

## 描述

set_response_header 过滤器用来设置请求响应的 Header 头信息。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: set_response_header
    filter:
      - set_response_header:
          headers:
            - Trial -> true
            - Department -> Engineering
```

## 参数说明

| 名称    | 类型 | 说明                                               |
| ------- | ---- | -------------------------------------------------- |
| headers | map  | 使用 `->` 作为标识符的键值对，用于设置 Header 信息 |
