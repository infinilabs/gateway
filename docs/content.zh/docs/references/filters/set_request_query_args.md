---
title: "set_request_query_args"
---

# set_request_query_args

## 描述

set_request_query_args 过滤器用来设置请求的 QueryString 参数信息。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: set_request_query_args
    filter:
      - set_request_query_args:
          args:
            - size -> 10
```

为避免

## 参数说明

| 名称 | 类型 | 说明                                                        |
| ---- | ---- | ----------------------------------------------------------- |
| args | map  | 使用 `->` 作为标识符的键值对，用于设置 QueryString 参数信息 |
