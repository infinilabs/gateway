---
title: "dump"
asciinema: true
---

# dump

## Description

The dump filter is used to dump relevant request information on terminals. It is mainly used for debugging.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: hello_world
    filter:
      - dump:
         request: true
         response: true
```

### Parameter Description

The dump filter is relatively simple. After the dump filter is inserted into a required flow handling phase, the terminal can output request information about the phase, facilitating debugging.

| Name            | Type  | Description                                              |
| --------------- | ----- | -------------------------------------------------------- |
| request         | bool  | Whether to output all complete request information       |
| response        | bool  | Whether to output all complete response information      |
| uri             | bool  | Whether to output the requested URI information          |
| query_args      | bool  | Whether to output the requested parameter information    |
| user            | bool  | Whether to output the requested user information         |
| api_key         | bool  | Whether to output the requested API key information      |
| request_header  | bool  | Whether to output the header information of the request  |
| response_header | bool  | Whether to output the header information of the response |
| status_code     | bool  | Whether to output the status code of the response        |
| context         | array | User-defined context information for output              |

### Outputting Context

You can use the `context` parameter to debug request context information. The following is an example of the configuration file.

```
flow:
  - name: echo
    filter:
      - set_response:
          status: 201
          content_type: "text/plain; charset=utf-8"
          body: "hello world"
      - set_response_header:
          headers:
            - Env -> Dev
      - dump:
          context:
            - _ctx.id
            - _ctx.tls
            - _ctx.remote_addr
            - _ctx.local_addr
            - _ctx.request.host
            - _ctx.request.method
            - _ctx.request.uri
            - _ctx.request.path
            - _ctx.request.body
            - _ctx.request.body_length
            - _ctx.request.query_args.from
            - _ctx.request.query_args.size
            - _ctx.request.header.Accept
            - _ctx.request.user
            - _ctx.response.status
            - _ctx.response.body
            - _ctx.response.content_type
            - _ctx.response.body_length
            - _ctx.response.header.Env
```

Start the gateway and run the following command:

```
curl http://localhost:8000/medcl/_search\?from\=1\&size\=100 -d'{search:query123}' -v -u 'medcl:123'
```

The gateway outputs the following information:

```
---- dumping context ----
_ctx.id  :  21474836481
_ctx.tls  :  false
_ctx.remote_addr  :  127.0.0.1:50925
_ctx.local_addr  :  127.0.0.1:8000
_ctx.request.host  :  localhost:8000
_ctx.request.method  :  POST
_ctx.request.uri  :  http://localhost:8000/medcl/_search?from=1&size=100
_ctx.request.path  :  /medcl/_search
_ctx.request.body  :  {search:query123}
_ctx.request.body_length  :  17
_ctx.request.query_args.from  :  1
_ctx.request.query_args.size  :  100
_ctx.request.header.Accept  :  */*
_ctx.request.user  :  medcl
_ctx.response.status  :  201
_ctx.response.body  :  hello world
_ctx.response.content_type  :  text/plain; charset=utf-8
_ctx.response.body_length  :  11
_ctx.response.header.Env  :  Dev
```
