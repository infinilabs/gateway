---
title: "set_response"
---

# set_response

## 描述

set_response 过滤器用来设置请求响应返回信息。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: set_response
    filter:
      - set_response:
          status: 200
          content_type: application/json
          body: '{"message":"hello world"}'
```

## 参数说明

| 名称         | 类型   | 说明                   |
| ------------ | ------ | ---------------------- |
| status       | int    | 请求状态码，默认 `200` |
| content_type | string | 设置请求返回的内容类型 |
| body         | string | 设置请求返回的结构体   |
