---
title: "request_host_limiter"
asciinema: true
---

# request_host_limiter

## Description

The request_host_limiter filter is used to control traffic based on the request host (domain name).

## Configuration Example

A configuration example is as follows:

```
flow:
  - name: rate_limit_flow
    filter:
      - request_host_limiter:
          host:
            - api.elasticsearch.cn:8000
            - logging.elasticsearch.cn:8000
          max_requests: 256
#          max_bytes: 102400 #100k
          action: retry # retry or drop
#          max_retry_times: 1000
#          retry_interval: 500 #100ms
          message: "you reached our limit"
```

The above configuration controls the traffic used for accessing domain names `api.elasticsearch.cn` and `logging.elasticsearch.cn` and the maximum allowable QPS is `256` per second.

## Parameter Description

| Name                 | Type   | Description                                                                                                                                                                                                                                              |
| -------------------- | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| host                 | array  | Host domain names that will participate in traffic control. If this parameter is not set, all host domain names will participate in traffic control. If an accessed domain name contains a port ID, add the port ID here. For example, `localhost:8080`. |
| interval             | string | Interval for evaluating whether traffic control conditions are met. The default value is `1s`.                                                                                                                                                           |
| max_requests         | int    | Maximum request count limit in the interval                                                                                                                                                                                                              |
| burst_requests       | int    | Burst request count limit in the interval                                                                                                                                                                                                                |
| max_bytes            | int    | Maximum request traffic limit in the interval                                                                                                                                                                                                            |
| burst_bytes          | int    | Burst request traffic limit in the interval                                                                                                                                                                                                              |
| action               | string | Processing action after traffic control is triggered. The value can be set as `retry` or `drop` and the default value is `retry`.                                                                                                                        |
| status               | string | Status code returned after traffic control conditions are met. The default value is `429`.                                                                                                                                                               |
| message              | string | Rejection message returned for a request, for which traffic control conditions are met                                                                                                                                                                   |
| retry_delay_in_ms    | int    | Interval for traffic control retry, in milliseconds. The default value is `10`.                                                                                                                                                                          |
| max_retry_times      | int    | Maximum retry count in the case of traffic control retries. The default value is `1000`.                                                                                                                                                                 |
| failed_retry_message | string | Rejection message returned for a request, for which the maximum retry count has been reached                                                                                                                                                             |
| log_warn_message     | bool   | Whether to log warn message                                                                                                                                                                                                                              |
