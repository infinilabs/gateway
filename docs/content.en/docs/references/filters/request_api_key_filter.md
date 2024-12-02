---
title: "request_api_key_filter"
---

# request_api_key_filter

## Description

When Elasticsearch conducts authentication through API keys, the request_api_key_filter is used to filter requests based on request API ID.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - request_api_key_filter:
          message: "Request filtered!"
          exclude:
            - VuaCfGcBCdbkQm-e5aOx
```

The above example shows that requests from `VuaCfGcBCdbkQm-e5aOx` will be rejected. See the following information.

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
< HTTP/1.1 403 Forbidden
< Server: INFINI
< Date: Mon, 12 Apr 2021 15:02:37 GMT
< content-type: text/plain; charset=utf-8
< content-length: 17
< FILTERED: true
< process: request_api_key_filter
<
* Connection #0 to host localhost left intact
{"error":true,"message":"Request filtered!"}%                                                              ➜  ~
```

## Parameter Description

| Name    | Type   | Description                                                                                                                              |
| ------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| exclude | array  | List of usernames, from which requests are refused to pass through                                                                       |
| include | array  | List of usernames, from which requests are allowed to pass through                                                                       |
| action  | string | Processing action after filtering conditions are met. The value can be set to `deny` or `redirect_flow` and the default value is `deny`. |
| status  | int    | Status code returned after the user-defined mode is matched                                                                              |
| message | string | Message text returned in user-defined `deny` mode                                                                                        |
| flow    | string | ID of the flow executed in user-defined `redirect_flow` mode                                                                             |

{{< hint info >}}
Note: If the `include` condition is met, requests are allowed to pass through only when at least one response code in `include` is met.
If only the `exclude` condition is met, any request that does not meet `exclude` is allowed to pass through.
{{< /hint >}}
