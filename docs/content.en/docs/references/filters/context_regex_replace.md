---
title: "context_regex_replace"
---

# context_regex_replace

## Description

The context_regex_replace filter is used to replace and modify relevant information in the request context by using regular expressions.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - context_regex_replace:
          context: "_ctx.request.path"
          pattern: "^/"
          to: "/cluster:"
          when:
            contains:
              _ctx.request.path: /_search
      - dump:
          request: true
```

This example replaces `curl localhost:8000/abc/_search` in requests with `curl localhost:8000/cluster:abc/_search`.

## Parameter Description

| Name    | Type   | Description                                          |
| ------- | ------ | ---------------------------------------------------- |
| context | string | Request context and corresponding key                |
| pattern | string | Regular expression used for matching and replacement |
| to      | string | Target string used for replacement                   |

A list of context variables that can be modified is provided below:

| Name                                 | Type   | Description                      |
| ------------------------------------ | ------ | -------------------------------- |
| \_ctx.request.uri                    | string | Complete URL of a request        |
| \_ctx.request.path                   | string | Request path                     |
| \_ctx.request.host                   | string | Request host                     |
| \_ctx.request.body                   | string | Request body                     |
| \_ctx.request.body_json.[JSON_PATH]  | string | Path to the JSON request object  |
| \_ctx.request.query_args.[KEY]       | string | URL query request parameter      |
| \_ctx.request.header.[KEY]           | string | Request header information       |
| \_ctx.response.header.[KEY]          | string | Response header information      |
| \_ctx.response.body                  | string | Returned response body           |
| \_ctx.response.body_json.[JSON_PATH] | string | Path to the JSON response object |
