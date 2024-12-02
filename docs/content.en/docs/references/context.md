---
title: "Request Context"
weight: 49
---

# Request Context

## What Is Context

Context is the entry for INFINI Gateway to access relevant information in the current running environment, such as the request source and configuration. You can use the `_ctx` keyword to access relevant fields, for example, `_ctx.request.uri`, which indicates the requested URL.

## Embedded Request Context

The embedded `_ctx` context objects of an HTTP request mainly include the following:

| Name        | Type   | Description                                     |
| ----------- | ------ | ----------------------------------------------- |
| id          | uint64 | Unique ID of the request                        |
| tls         | bool   | Whether the request is a TLS request            |
| remote_ip   | string | Source IP of the client                         |
| remote_addr | string | Source IP address of the client, including port |
| local_ip    | string | Gateway local IP address                        |
| local_addr  | string | Gateway local IP address, including port        |
| elapsed     | int64  | Time that the request has been executed (ms)    |
| request.\*  | object | Request description                             |
| response.\* | object | Response description                            |

### request

The `request` object has the following attributes:

| Name        | Type   | Description                                |
| ----------- | ------ | ------------------------------------------ |
| to_string   | string | Complete HTTP request in text form         |
| host        | string | Accessed destination host name/domain name |
| method      | string | Request type                               |
| uri         | string | Complete URL of request                    |
| path        | string | Request path                               |
| query_args  | map    | URL request parameter                      |
| username    | string | Name of the user who initiates the request |
| password    | string | Password of the user                       |
| header      | map    | Header parameter                           |
| body        | string | Request body                               |
| body_json   | object | JSON request body object                   |
| body_length | int    | Request body length                        |

If the request body data submitted by the client is in JSON format, you can use `body_json` to access the data. See the following example.

```
curl -u tesla:password -XGET "http://localhost:8000/medcl/_search?pretty" -H 'Content-Type: application/json' -d'
{
  "query":{
	"bool":{
	"must":[{"match":{"name":"A"}},{"match":{"age":18}}]
	}
},
"size":900,
  "aggs": {
    "total_num": {
      "terms": {
        "field": "name1",
        "size": 1000000
      }
    }
  }
}'
```

In JSON data, `.` is used to identify the path. If the data is an array, you can use `[Subscript]` to access a specified element, for example, you can use a `dump` filter for debugging as follows:

```
  - name: cache_first
    filter:
      - dump:
          context:
            - _ctx.request.body_json.size
            - _ctx.request.body_json.aggs.total_num.terms.field
            - _ctx.request.body_json.query.bool.must.[1].match.age
```

The output is as follows:

```
_ctx.request.body_json.size  :  900
_ctx.request.body_json.aggs.total_num.terms.field  :  name1
_ctx.request.body_json.query.bool.must.[1].match.age  :  18
```

### response

The `response` object has the following attributes:

| Name         | Type   | Description                         |
| ------------ | ------ | ----------------------------------- |
| to_string    | string | Complete HTTP response in text form |
| status       | int    | Request status code                 |
| header       | map    | Header parameter                    |
| content_type | string | Response body type                  |
| body         | string | Response body                       |
| body_json    | object | JSON request body object            |
| body_length  | int    | Response body length                |

## System Context

The `_sys.*` object has the following attributes:

| Name           | Type   | Description                                       |
| -------------- | ------ | ------------------------------------------------- |
| hostname       | string | Hostname of the gateway deployed                  |
| month_of_now   | int    | Month of now, range from `[1,12]`                 |
| weekday_of_now | int    | Weekday of now, range from `[0,6]`, `0` is Sunday |
| day_of_now     | int    | Day of now                                        |
| hour_of_now    | int    | Hour of now, range from `[0,23]`                  |
| minute_of_now  | int    | Minute of now, range from `[0,59]`                |
| second_of_now  | int    | Second of now, range from `[0,59]`                |
| unix_timestamp_of_now        | int64    | Unix timestamp of the current time      |
| unix_timestamp_milli_of_now  | int64    | Unix timestamp of the current time in milliseconds |

## Utility Context

These are utility context can ube used for quickly obtaining relevant parameters. The object `_util.*` has the following properties:

| Name            | Type    | Description                                 |
| --------------- | ------- | ------------------------------------------- |
| generate_uuid   | string  | Retrieve a random UUID string parameter    |
| increment_id    | string  | Retrieve an auto-incrementing numeric identifier. Default bucket is `default`, customize bucket are supported, e.g., `_util.increment_id.mybucket` |