---
title: "set_context"
---

# set_context

## Description

The set_context filter is used to set relevant information for the request context.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - set_response:
          body: '{"message":"hello world"}'
      - set_context:
          context:
#            _ctx.request.uri: http://baidu.com
#            _ctx.request.path: new_request_path
#            _ctx.request.host: api.infinilabs.com
#            _ctx.request.method: DELETE
#            _ctx.request.body: "hello world"
#            _ctx.request.body_json.explain: true
#            _ctx.request.query_args.from: 100
#            _ctx.request.header.ENV: dev
#            _ctx.response.content_type: "application/json"
#            _ctx.response.header.TIMES: 100
#            _ctx.response.status: 419
#            _ctx.response.body: "new_body"
            _ctx.response.body_json.success: true
      - dump:
          request: true
```

## Parameter Description

| Name    | Type | Description                             |
| ------- | ---- | --------------------------------------- |
| context | map  | Request context and corresponding value |

A list of supported context variables is provided below:

| Name                                 | Type   | Description                      |
| ------------------------------------ | ------ | -------------------------------- |
| \_ctx.request.uri                    | string | Complete URL of a request        |
| \_ctx.request.path                   | string | Request path                     |
| \_ctx.request.host                   | string | Request host                     |
| \_ctx.request.method                 | string | Request method type              |
| \_ctx.request.body                   | string | Request body                     |
| \_ctx.request.body_json.[JSON_PATH]  | string | Path to the JSON request object  |
| \_ctx.request.query_args.[KEY]       | string | URL query request parameter      |
| \_ctx.request.header.[KEY]           | string | Request header information       |
| \_ctx.response.content_type          | string | Request body type                |
| \_ctx.response.header.[KEY]          | string | Response header information      |
| \_ctx.response.status                | int    | Returned status code             |
| \_ctx.response.body                  | string | Returned response body           |
| \_ctx.response.body_json.[JSON_PATH] | string | Path to the JSON response object |
