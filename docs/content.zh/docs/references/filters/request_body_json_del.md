---
title: "request_body_json_del"
---

# request_body_json_del

## 描述

request_body_json_del 过滤器用来删除 JSON 格式的请求体里面的部分字段。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - request_body_json_del:
          path:
          - query.bool.should.[0]
          - query.bool.must
```

## 参数说明

| 名称           | 类型  | 说明                                                  |
| -------------- | ----- | ----------------------------------------------------- |
| path           | array | 需要删除的 JSON PATH 键值                             |
| ignore_missing | bool  | 如果这个 JSON Path 不存在，是否忽略处理，默认 `false` |
