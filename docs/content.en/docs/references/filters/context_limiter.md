---
title: "context_limiter"
---

# context_limiter

## Description

The context_limiter filter is used to control the traffic based on request context.

## Configuration Example

A configuration example is as follows:

```
flow:
  - name: default_flow
    filter:
      - context_limiter:
          max_requests: 1
          action: drop
          context:
            - _ctx.request.path
            - _ctx.request.header.Host
            - _ctx.request.header.Env
```

The above configuration combines three context variables (`_ctx.request.path`, `_ctx.request.header.Host`, and `_ctx.request.header.Env`) into a bucket for traffic control.
The allowable maximum queries per second (QPS) is `1` per second. Subsequent requests out of the traffic control range are directly denied.

## Parameter Description

| Name                 | Type   | Description                                                                                                                       |
| -------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------- |
| context              | array  | Context variables, which form a bucket key                                                                                        |
| interval             | string | Interval for evaluating whether traffic control conditions are met. The default value is `1s`.                                    |
| max_requests         | int    | Maximum request count limit in the interval                                                                                       |
| burst_requests       | int    | Burst request count limit in the interval                                                                                         |
| max_bytes            | int    | Maximum request traffic limit in the interval                                                                                     |
| burst_bytes          | int    | Burst request traffic limit in the interval                                                                                       |
| action               | string | Processing action after traffic control is triggered. The value can be set as `retry` or `drop` and the default value is `retry`. |
| status               | string | Status code returned after traffic control conditions are met. The default value is `429`.                                        |
| message              | string | Rejection message returned for a request, for which traffic control conditions are met                                            |
| retry_delay_in_ms    | int    | Interval for traffic control retry, in milliseconds. The default value is `10`.                                                   |
| max_retry_times      | int    | Maximum retry count in the case of traffic control retries. The default value is `1000`.                                          |
| failed_retry_message | string | Rejection message returned for a request, for which the maximum retry count has been reached                                      |
| log_warn_message     | bool   | Whether to log warn message                                                                                                       |
