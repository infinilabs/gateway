---
title: "request_path_limiter"
asciinema: true
---

# request_path_limiter

## 描述

request_path_limiter 过滤器用来定义请求的限速规则，可以实现索引级别的限速。

## 配置示例

配置示例如下：

```
flow:
  - name: rate_limit_flow
    filter:
      - request_path_limiter:
          message: "Hey, You just reached our request limit!"
          rules:
            - pattern: "/(?P<index_name>medcl)/_search"
              max_qps: 3
              group: index_name
            - pattern: "/(?P<index_name>.*?)/_search"
              max_qps: 100
              group: index_name
```

上面的配置中，对 `medcl` 这个索引执行查询，允许的最大 qps 为 `3`，而对其它的索引执行查询的 qps 为 `100`。

## 参数说明

| 名称          | 类型   | 说明                                                                                            |
| ------------- | ------ | ----------------------------------------------------------------------------------------------- |
| message       | string | 设置达到限速条件的请求的返回消息                                                                |
| rules         | array  | 设置限速的策略，支持多种规则，按照配置的先后顺序处理，先匹配的先执行                            |
| rules.pattern | string | 使用正则表达式来对 URL 的 Path 进行规则匹配，必须提供一个 group 名称，用于作为限速的 bucket key |
| rules.group   | string | 正则表达式里面定义的 group 名称，将用于请求次数的统计，相同的 group 值视为一类请求              |
| rules.max_qps | int    | 定义每组请求的最大的 qps 参数，超过该值将触发限速行为                                           |
