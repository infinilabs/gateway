---
title: "response_status_filter"
asciinema: true
---

# response_status_filter

## Description

The response_status_filter is used to filter traffic based on the status code responded by the back-end service.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - response_status_filter:
          message: "Request filtered!"
          exclude:
            - 404
          include:
            - 200
            - 201
            - 500
```

## Parameter Description

| Name    | Type   | Description                                                                                                                              |
| ------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| exclude | array  | Response code for refusing to allow traffic to pass through                                                                              |
| include | array  | Response code for allowing traffic to pass through                                                                                       |
| action  | string | Processing action after filtering conditions are met. The value can be set to `deny` or `redirect_flow` and the default value is `deny`. |
| status  | int    | Status code returned after the user-defined mode is matched                                                                              |
| message | string | Message text returned in user-defined `deny` mode                                                                                        |
| flow    | string | ID of the flow executed in user-defined `redirect_flow` mode                                                                             |

{{< hint info >}}
Note: If the `include` condition is met, requests are allowed to pass through only when at least one response code in `include` is met.
If only the `exclude` condition is met, any request that does not meet `exclude` is allowed to pass through.
{{< /hint >}}
