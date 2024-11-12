---
title: "bulk_request_throttle"
---

# bulk_request_throttle

## Description

bulk_request_throttle 过滤器用来对 Elasticsearch 的 Bulk 请求进行限速。

## Configuration Example

A simple example is as follows:

```
flow:
  - name: bulk_request_mutate
    filter:
      - bulk_request_throttle:
          indices:
            test:
              max_requests: 5
              action: drop
              message: "test writing too fast。"
              log_warn_message: true
            filebeat-*:
              max_bytes: 512
              action: drop
              message: "filebeat indices writing too fast。"
              log_warn_message: true
```

## Parameter Description

| Name                                | Type   | Description                                                                                                                       |
| ----------------------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------- |
| indices                             | map    | The indices which wanted to throttle                                                                                              |
| indices.[NAME].interval             | string | 评估限速的单位时间间隔，默认为 `1s`                                                                                               |
| indices.[NAME].max_requests         | int    | Maximum request count limit in the interval                                                                                       |
| indices.[NAME].burst_requests       | int    | Burst request count limit in the interval                                                                                         |
| indices.[NAME].max_bytes            | int    | Maximum request traffic limit in the interval                                                                                     |
| indices.[NAME].burst_bytes          | int    | Burst request traffic limit in the interval                                                                                       |
| indices.[NAME].action               | string | Processing action after traffic control is triggered. The value can be set as `retry` or `drop` and the default value is `retry`. |
| indices.[NAME].status               | string | Status code returned after traffic control conditions are met. The default value is `429`.                                        |
| indices.[NAME].message              | string | Rejection message returned for a request, for which traffic control conditions are met                                            |
| indices.[NAME].retry_delay_in_ms    | int    | Interval for traffic control retry, in milliseconds. The default value is `10`.                                                   |
| indices.[NAME].max_retry_times      | int    | Maximum retry count in the case of traffic control retries. The default value is `1000`.                                          |
| indices.[NAME].failed_retry_message | string | Rejection message returned for a request, for which the maximum retry count has been reached                                      |
| indices.[NAME].log_warn_message     | bool   | Whether to log warn message                                                                                                       |
