---
title: "request_path_limiter"
asciinema: true
---

# request_path_limiter

## Description

The request_path_limiter filter is used to define traffic control rules for requests. It can implement index-level traffic control.

## Configuration Example

A configuration example is as follows:

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

In the above configuration, the query is performed against the `medcl` query, the allowable maximum QPS is `3`, and the QPS is `100` for queries performed against other indexes.

## Parameter Description

| Name          | Type   | Description                                                                                                                                                                                   |
| ------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| message       | string | Message returned for a request, for which traffic control conditions are met                                                                                                                  |
| rules         | array  | Traffic control rule. Multiple rules can be configured, which are matched based on their configuration sequence. If a rule is matched earlier, the corresponding action is performed earlier. |
| rules.pattern | string | Regular expression rule used for URL path matching. One group name must be provided as the bucket key for traffic control.                                                                    |
| rules.group   | string | Group name defined in the regular expression, which is used to count the number of requests. Requests with the same group value are regarded as the same type of request.                     |
| rules.max_qps | int    | Maximum QPS defined for each group of requests. When the actual value exceeds this value, the traffic control action is triggered.                                                            |
