---
title: "context_filter"
---

# context_filter

## Description

The context_filter is used to filter traffic by request context.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - context_filter:
          context: _ctx.request.path
          message: "request not allowed."
          status: 403
          must: #must match all rules to continue
            prefix:
              - /medcl
            contain:
              - _search
            suffix:
              - _search
            wildcard:
              - /*/_search
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
              - _refresh
            suffix:
              - _count
              - _refresh
            wildcard:
              - /*/_refresh
            regex:
              - ^/\.m[\w]+dcl
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
| context     | string | Context variable                                                                                                                         |
| exclude     | array  | List of variables used to refuse requests to pass through                                                                                |
| include     | array  | List of variables used to allow requests to pass through                                                                                 |
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
