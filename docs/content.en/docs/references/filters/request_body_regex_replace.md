---
title: "request_body_regex_replace"
---

# request_body_regex_replace

## Description

The request_body_regex_replace filter is used to replace string content in a request body by using a regular expression.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - request_body_regex_replace:
          pattern: '"size": 10000'
          to: '"size": 100'
      - elasticsearch:
          elasticsearch: prod
      - dump:
          request: true
```

The above example changes the size from 10000 to 100 in the request body sent to Elasticsearch. The filter can be used to dynamically fix errors or incorrect queries.

The test is as follows:

```
curl -XPOST "http://localhost:8000/myindex/_search" -H 'Content-Type: application/json' -d'
{
  "query": {
    "match_all": {}
  },"size": 10000
}'
```

The actual query is as follows:

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

| Name    | Type   | Description                                          |
| ------- | ------ | ---------------------------------------------------- |
| pattern | string | Regular expression used for matching and replacement |
| to      | string | Target string used for replacement                   |
