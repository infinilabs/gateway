---
title: "request_client_ip_limiter"
asciinema: true
---

# request_client_ip_limiter

## Description

The request_client_ip_limiter filter is used to control traffic based on the request client IP address.

## Configuration Example

A configuration example is as follows:

```
flow:
  - name: rate_limit_flow
    filter:
      - request_client_ip_limiter:
          ip: #only limit for specify ips
            - 127.0.0.1
          max_requests: 256
#          max_bytes: 102400 #100k
          action: retry # retry or drop
#          max_retry_times: 1000
#          retry_interval: 500 #100ms
          message: "your ip reached our limit"
```

The above configuration controls the traffic with the IP address of `127.0.0.1` and the allowable maximum QPS is `256` per second.

## Parameter Description

| Name                 | Type   | Description                                                                                                                                       |
| -------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| ip                   | array  | Client IP addresses that will participate in traffic control. If this parameter is not set, all IP addresses will participate in traffic control. |
| interval             | string | Interval for evaluating whether traffic control conditions are met. The default value is `1s`.                                                    |
| max_requests         | int    | Maximum request count limit in the interval                                                                                                       |
| burst_requests       | int    | Burst request count limit in the interval                                                                                                         |
| max_bytes            | int    | Maximum request traffic limit in the interval                                                                                                     |
| burst_bytes          | int    | Burst request traffic limit in the interval                                                                                                       |
| action               | string | Processing action after traffic control is triggered. The value can be set as `retry` or `drop` and the default value is `retry`.                 |
| status               | string | Status code returned after traffic control conditions are met. The default value is `429`.                                                        |
| message              | string | Rejection message returned for a request, for which traffic control conditions are met                                                            |
| retry_delay_in_ms    | int    | Interval for traffic control retry, in milliseconds. The default value is `10`.                                                                   |
| max_retry_times      | int    | Maximum retry count in the case of traffic control retries. The default value is `1000`.                                                          |
| failed_retry_message | string | Rejection message returned for a request, for which the maximum retry count has been reached                                                      |
| log_warn_message     | bool   | Whether to log warn message                                                                                                                       |
