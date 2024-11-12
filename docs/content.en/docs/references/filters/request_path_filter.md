---
title: "request_path_filter"
asciinema: true
---

# request_path_filter

## Description

The request_path_filter is used to filter traffic based on request path.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - request_path_filter:
          must: #must match all rules to continue
            prefix:
              - /medcl
            contain:
              - _search
            suffix:
              - _count
              - _refresh
            wildcard:
              - /*/_refresh
            regex:
              - ^/m[\w]+dcl
          must_not: # any match will be filtered
            prefix:
              - /.kibana
              - /_security
              - /_security
              - /gateway_requests*
              - /.reporting
              - /_monitoring/bulk
            contain:
              - _search
            suffix:
              - _count
              - _refresh
            wildcard:
              - /*/_refresh
            regex:
              - ^/m[\w]+dcl
          should:
            prefix:
              - /medcl
            contain:
              - _search
              - _async_search
            suffix:
              - _refresh
            wildcard:
              - /*/_refresh
            regex:
              - ^/m[\w]+dcl
```

## Parameter Description

| Name        | Type   | Description                                                                                                                              |
| ----------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| must.\*     | object | Requests are allowed to pass through only when all conditions are met.                                                                   |
| must_not.\* | object | Requests are allowed to pass through only when none of the conditions are met.                                                           |
| should.\*   | object | Requests are allowed to pass through when any condition is met.                                                                          |
| \*.prefix   | array  | Whether a request begins with a specific character                                                                                       |
| \*.suffix   | array  | Whether a request ends with a specific character                                                                                         |
| \*.contain  | array  | Whether a request contains a specific character                                                                                          |
| \*.wildcard | array  | Whether a request meets pattern matching rules                                                                                           |
| \*.regex    | array  | Whether a request meets regular expression matching rules                                                                                |
| action      | string | Processing action after filtering conditions are met. The value can be set to `deny` or `redirect_flow` and the default value is `deny`. |
| status      | int    | Status code returned after the user-defined mode is matched                                                                              |
| message     | string | Message text returned in user-defined `deny` mode                                                                                        |
| flow        | string | ID of the flow executed in user-defined `redirect_flow` mode                                                                             |

Note: If only the `should` condition is met, requests are allowed to pass through only when at least one item in `should` is met.
