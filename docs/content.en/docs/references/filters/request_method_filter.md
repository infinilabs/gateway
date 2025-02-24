---
title: "request_method_filter"
asciinema: true
---

# request_method_filter

## Description

The request_method_filter is used to filter traffic based on request method.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - request_method_filter:
          exclude:
            - PUT
            - POST
          include:
            - GET
            - HEAD
            - DELETE
```

## Parameter Description

| Name    | Type   | Description                                                                                                                              |
| ------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| exclude | array  | Methods of requests that are refused to pass through                                                                                     |
| include | array  | Methods of requests that are allowed to pass through                                                                                     |
| action  | string | Processing action after filtering conditions are met. The value can be set to `deny` or `redirect_flow` and the default value is `deny`. |
| status  | int    | Status code returned after the user-defined mode is matched                                                                              |
| message | string | Message text returned in user-defined `deny` mode                                                                                        |
| flow    | string | ID of the flow executed in user-defined `redirect_flow` mode                                                                             |

{{< hint info >}}
Note: If the `include` condition is met, requests are allowed to pass through only when at least one response code in `include` is met.
If only the `exclude` condition is met, any request that does not meet `exclude` is allowed to pass through.
{{< /hint >}}
