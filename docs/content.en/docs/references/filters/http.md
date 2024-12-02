---
title: "http"
---

# http

## Description

The http filter is used to forward requests to a specified HTTP server as a proxy.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: default_flow
    filter:
      - basic_auth:
          valid_users:
            medcl: passwd
      - http:
          schema: "http" #https or http
          #host: "192.168.3.98:5601"
          hosts:
           - "192.168.3.98:5601"
           - "192.168.3.98:5602"
```

## Parameter Description

| Name                     | Type     | Description                                                                  |
| ------------------------ | -------- | ---------------------------------------------------------------------------- |
| schema                   | string   | `http` or `https`                                                            |
| host                     | string   | Target host address containing the port ID, for example, `localhost:9200`    |
| hosts                    | array    | Host address list. The addresses are tried in sequence after a fault occurs. |
| skip_failure_host        | bool     | Skip hosts in failure, default `true`                                        |
| max_connection_per_node  | int      | The max connections per node, default `5000`                                 |
| max_response_size        | int      | The max length of response supported                                         |
| max_retry_times          | int      | The max num of retries, default `0`                                          |
| retry_delay_in_ms        | int      | The latency before next retry in millisecond, default `1000`                 |
| skip_cleanup_hop_headers | bool     | Remove Hop-by-hop Headers                                                    |
| max_conn_wait_timeout    | duration | The max time wait to create new connections, default `30s`                   |
| max_idle_conn_duration   | duration | The max duration of idle connections, default `30s`                          |
| max_conn_duration        | duration | The max duration of keepalived connections, default `0s`                     |
| timeout                  | duration | Request timeout duration, default `30s`                                      |
| read_timeout             | duration | Read request timeout duration, default `0s`                                  |
| write_timeout            | duration | Write request timeout duration, default `0s`                                 |
| read_buffer_size         | int      | Read buffer size, default `16384`                                            |
| write_buffer_size        | int      | Write buffer size, default `16384`                                           |
| tls_insecure_skip_verify | bool     | Skip the TLS verification, default `true`                                    |
