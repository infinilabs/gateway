---
title: "logging"
asciinema: true
---

# logging

## 描述

logging 过滤器用来按请求记录下来，通过异步记录到本地磁盘的方式，尽可能降低对请求的延迟影响，对于流量很大的场景，建议配合其它请求过滤器来降低日志的总量。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: test
    filter:
      - logging:
          queue_name: request_logging
```

记录的请求日志样例如下：

```
 {
        "_index" : "gateway_requests",
        "_type" : "doc",
        "_id" : "EH5bG3gBsbC2s3iWFzCF",
        "_score" : 1.0,
        "_source" : {
          "tls" : false,
          "@timestamp" : "2021-03-10T08:57:30.645Z",
          "conn_time" : "2021-03-10T08:57:30.635Z",
          "flow" : {
            "from" : "127.0.0.1",
            "process" : [
              "request_body_regex_replace",
              "get_cache",
              "date_range_precision_tuning",
              "get_cache",
              "elasticsearch",
              "set_cache",
              "||",
              "request_logging"
            ],
            "relay" : "192.168.43.101-Quartz",
            "to" : [
              "localhost:9200"
            ]
          },
          "id" : 3,
          "local_ip" : "127.0.0.1",
          "remote_ip" : "127.0.0.1",
          "request" : {
            "body_length" : 53,
            "body" : """

{
  "query": {
    "match_all": {}
  },"size": 100
}
""",
            "header" : {
              "content-type" : "application/json",
              "User-Agent" : "curl/7.54.0",
              "Accept" : "*/*",
              "Host" : "localhost:8000",
              "content-length" : "53"
            },
            "host" : "localhost:8000",
            "local_addr" : "127.0.0.1:8000",
            "method" : "POST",
            "path" : "/myindex/_search",
            "remote_addr" : "127.0.0.1:63309",
            "started" : "2021-03-10T08:57:30.635Z",
            "uri" : "http://localhost:8000/myindex/_search"
          },
          "response" : {
            "body_length" : 441,
            "cached" : false,
            "elapsed" : 9.878,
            "status_code" : 200,
            "body" : """{"took":0,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":1,"max_score":1.0,"hits":[{"_index":"myindex","_type":"doc","_id":"c132mhq3r0otidqkac1g","_score":1.0,"_source":{"name":"local","enabled":true,"endpoint":"http://localhost:9200","basic_auth":{},"discovery":{"refresh":{}},"created":"2021-03-08T21:48:55.687557+08:00","updated":"2021-03-08T21:48:55.687557+08:00"}}]}}""",
            "header" : {
              "UPSTREAM" : "localhost:9200",
              "process" : "request_body_regex_replace->get_cache->date_range_precision_tuning->get_cache->elasticsearch->set_cache",
              "content-length" : "441",
              "content-type" : "application/json; charset=UTF-8",
              "Server" : "INFINI",
              "CLUSTER" : "dev"
            },
            "local_addr" : "127.0.0.1:63310"
          }
        }
      }
```

## 参数说明

| 名称                   | 类型   | 说明                                                             |
| ---------------------- | ------ | ---------------------------------------------------------------- |
| queue_name             | string | 将请求日志保存的本地磁盘的队列名称                               |
| format_header_keys     | bool   | 是否将 Header 标准化，都转成小写，默认 `false`                   |
| remove_authorization   | bool   | 是否将 Authorization 信息从 Header 里面移除，默认 `true`         |
| max_request_body_size  | int    | 是否将过长的请求消息进行截断，默认 `1024` ，即保留 1024 个字符   |
| max_response_body_size | int    | 是否将过长的返回消息进行截断，默认 `1024` ，即保留 1024 个字符   |
| min_elapsed_time_in_ms | int    | 按照请求的响应时间进行过滤，最低超过多少 ms 的请求才会被记录下来 |
| bulk_stats_details     | bool   | 是否记录 bulk 请求详细的按照索引的统计信息，默认 `true`          |
