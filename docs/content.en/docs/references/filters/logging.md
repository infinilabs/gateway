---
title: "logging"
asciinema: true
---

# logging

## Description

The logging filter is used to asynchronously record requests to the local disk to minimize the delay of requests. In scenarios with heavy traffic, you are advised to use other request filters jointly to reduce the total number of logs.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - logging:
          queue_name: request_logging
```

An example of a recorded request log is as follows:

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

## Parameter Description

| Name                   | Type   | Description                                                                                                                                               |
| ---------------------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| queue_name             | string | Name of a queue that stores request logs in the local disk                                                                                                |
| format_header_keys     | bool   | Whether to standardize the header and convert it into lowercase letters. The default value is `false`.                                                    |
| remove_authorization   | bool   | Whether to remove authorization information from the header. The default value is `true`.                                                                 |
| max_request_body_size  | int    | Whether to truncate a very long request message. The default value is `1024`, indicating that 1024 characters are retained.                               |
| max_response_body_size | int    | Whether to truncate a very long response message. The default value is `1024`, indicating that 1024 characters are retained.                              |
| min_elapsed_time_in_ms | int    | Request filtering based on response time, that is, the minimum time (ms) for request logging. A request with time that exceeds this value will be logged. |
| bulk_stats_details     | bool   | Whether to record detailed index-based bulk request statistics. The default value is `true`.                                                              |
