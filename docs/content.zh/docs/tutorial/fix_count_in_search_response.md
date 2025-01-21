---
title: "兼容不同版本的响应 Count 结构"
weight: 100
---

# 兼容不同版本的查询响应结果的 Count 结构

Elasticsearch 在 7.0 之后的版本中，为了优化性能，搜索结果的命中数默认不进行精确的计数统计，同时对搜索结果的响应体进行了调整，
这样势必会造成已有代码的不兼容，如何快速修复呢？

## 结构对比

首先来对比下前后差异：

7 之前的搜索结构如下，`total` 显示的具体的数值：

```
{
  "took": 53,
  "timed_out": false,
  "_shards": {
    "total": 1,
    "successful": 1,
    "skipped": 0,
    "failed": 0
  },
  "hits": {
    "total": 0,
    "max_score": null,
    "hits": []
  }
}
```

7 之后的搜索结构如下，`total` 变成了一组描述范围的对象：

```
{
  "took": 3,
  "timed_out": false,
  "_shards": {
    "total": 1,
    "successful": 1,
    "skipped": 0,
    "failed": 0
  },
  "hits": {
    "total": {
      "value": 10000,
      "relation": "gte"
    },
    "max_score": 1,
    "hits": []
  }
}
```

## Elasticsearch 提供的参数

不过在 7 里面，Elasticsearch 也提供了一个参数来控制是否进行精确计数，通过在查询请求的 url 参数里面加上 `rest_total_hits_as_int=true` 即可使用旧的行为方式，默认未开启。

文档链接：https://www.elastic.co/guide/en/elasticsearch/reference/current/search-search.html

不过需要修改程序来添加这个参数，可能需要调整后端代码和前端分页及展示的相关，改动量可能不小。

## 使用极限网关来快速修复

如果不希望修改程序，可以使用极限网关来快速修复相应的查询，并主动为搜索查询添加相应的查询参数，同时还可以限定为哪些请求来源进行添加，
比如，只对特定的业务调用方来进行调整，这里以 `curl` 命令来进行举例，只对来自 `curl` 调试的查询进行添加，示例如下：

```
entry:
 - name: es_entrypoint
   enabled: true
   router: default
   network:
    binding: 0.0.0.0:8000

router:
 - name: default
   default_flow: main_flow

flow:
 - name: main_flow
   filter:
    - set_request_query_args:
       args:
        - rest_total_hits_as_int -> true
       when:
         and:
           - contains:
               _ctx.request.path: "_search"
           - equals:
               _ctx.request.header.User-Agent: "curl/7.54.0"
    - record:
        stdout: true
    - elasticsearch:
       elasticsearch: es-server
    - dump:
        response_body: true

elasticsearch:
 - name: es-server
   enabled: true
   endpoints:
    - http://192.168.3.188:9206

```

最后效果如下：

{{% load-img "/img/fix-search-response-count.png" "" %}}

如图 `1` 表示走浏览器访问网关的搜索结果，`2` 表示走命令行 curl 命令返回的搜索结果，其中通过 `User-Agent` 头信息可以匹配到 curl 命令，同时只对搜索条件附加参数，避免影响其他的请求。
