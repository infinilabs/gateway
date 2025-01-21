---
title: "request_body_json_set"
---

# request_body_json_set

## 描述

request_body_json_set 过滤器用来修改 JSON 格式的请求体。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - request_body_json_set:
          path:
          - aggs.total_num.terms.field -> "name"
          - aggs.total_num.terms.size -> 3
          - size -> 0
```

## 参数说明

| 名称           | 类型 | 说明                                                    |
| -------------- | ---- | ------------------------------------------------------- |
| path           | map  | 使用 `->` 作为标识符的键值对， JSON PATH 和需要替换的值 |
| ignore_missing | bool | 如果这个 JSON Path 不存在，是否忽略处理，默认 `false`   |
