---
title: "request_client_ip_filter"
---

# request_client_ip_filter

## Description

The request_client_ip_filter is used to filter traffic based on source user IP addresses of requests.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - request_client_ip_filter:
          exclude:
          - 192.168.3.67
```

The above example shows that requests from `192.168.3.67` are not allowed to pass through.

The following is an example of route redirection.

```
flow:
  - name: echo
    filter:
      - echo:
          message: hello stanger
  - name: default_flow
    filter:
      - request_client_ip_filter:
          action: redirect_flow
          flow: echo
          exclude:
            - 192.168.3.67
```

Requests from `192.168.3.67` are redirected to another `echo` flow.

## Parameter Description

| Name    | Type   | Description                                                                                                                              |
| ------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| exclude | array  | List of IP arrays, from which requests are refused to pass through                                                                       |
| include | array  | List of IP arrays, from which requests are allowed to pass through                                                                       |
| action  | string | Processing action after filtering conditions are met. The value can be set to `deny` or `redirect_flow` and the default value is `deny`. |
| status  | int    | Status code returned after the user-defined mode is matched                                                                              |
| message | string | Message text returned in user-defined `deny` mode                                                                                        |
| flow    | string | ID of the flow executed in user-defined `redirect_flow` mode                                                                             |

{{< hint info >}}
Note: If the `include` condition is met, requests are allowed to pass through only when at least one response code in `include` is met.
If only the `exclude` condition is met, any request that does not meet `exclude` is allowed to pass through.
{{< /hint >}}
