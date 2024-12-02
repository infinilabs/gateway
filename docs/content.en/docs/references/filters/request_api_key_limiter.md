---
title: "request_api_key_limiter"
asciinema: true
---

# request_api_key_limiter

## Description

The request_api_key_limiter filter is used to control traffic by API key.

## Configuration Example

A configuration example is as follows:

```
flow:
  - name: rate_limit_flow
    filter:
     - request_api_key_limiter:
         id:
           - VuaCfGcBCdbkQm-e5aOx
         max_requests: 1
         action: drop # retry or drop
         message: "your api_key reached our limit"
```

The above configuration controls the traffic with the API ID of `VuaCfGcBCdbkQm-e5aOx` and the allowable maximum QPS is `1` per second.

```
➜  ~ curl localhost:8000 -H "Authorization: ApiKey VnVhQ2ZHY0JDZGJrUW0tZTVhT3g6dWkybHAyYXhUTm1zeWFrdzl0dk5udw==" -v
* Rebuilt URL to: localhost:8000/
*   Trying 127.0.0.1...
* TCP_NODELAY set
* Connected to localhost (127.0.0.1) port 8000 (#0)
> GET / HTTP/1.1
> Host: localhost:8000
> User-Agent: curl/7.54.0
> Accept: */*
> Authorization: ApiKey VnVhQ2ZHY0JDZGJrUW0tZTVhT3g6dWkybHAyYXhUTm1zeWFrdzl0dk5udw==
>
< HTTP/1.1 429 Too Many Requests
< Server: INFINI
< Date: Mon, 12 Apr 2021 15:14:52 GMT
< content-type: text/plain; charset=utf-8
< content-length: 30
< process: request_api_key_limiter
<
* Connection #0 to host localhost left intact
your api_key reached our limit%
```

## Parameter Description

| Name                 | Type   | Description                                                                                                                           |
| -------------------- | ------ | ------------------------------------------------------------------------------------------------------------------------------------- |
| id                   | array  | IDs of APIs that will participate in traffic control. If this parameter is not set, all API keys will participate in traffic control. |
| interval             | string | Interval for evaluating whether traffic control conditions are met. The default value is `1s`.                                        |
| max_requests         | int    | Maximum request count limit in the interval                                                                                           |
| burst_requests       | int    | Burst request count limit in the interval                                                                                             |
| max_bytes            | int    | Maximum request traffic limit in the interval                                                                                         |
| burst_bytes          | int    | Burst request traffic limit in the interval                                                                                           |
| action               | string | Processing action after traffic control is triggered. The value can be set as `retry` or `drop` and the default value is `retry`.     |
| status               | string | Status code returned after traffic control conditions are met. The default value is `429`.                                            |
| message              | string | Rejection message returned for a request, for which traffic control conditions are met                                                |
| retry_delay_in_ms    | int    | Interval for traffic control retry, in milliseconds. The default value is `10`.                                                       |
| max_retry_times      | int    | Maximum retry count in the case of traffic control retries. The default value is `1000`.                                              |
| failed_retry_message | string | Rejection message returned for a request, for which the maximum retry count has been reached                                          |
| log_warn_message     | bool   | Whether to log warn message                                                                                                           |
