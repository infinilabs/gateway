---
title: "dump_hash"
---

# dump_hash

## 描述

dump_hash 处理器用来导出集群的索引文档并计算 Hash。

## 配置示例

一个简单的示例如下：

```
pipeline:
- name: bulk_request_ingest
  auto_start: true
  keep_running: true
  processor:
    - dump_hash: #dump es1's doc
        indices: "medcl-dr3"
        scroll_time: "10m"
        elasticsearch: "source"
        query: "field1:elastic"
        fields: "doc_hash"
        output_queue: "source_docs"
        batch_size: 10000
        slice_size: 5
```

## 参数说明

| 名称                 | 类型   | 说明                                                                                                         |
| -------------------- | ------ | ------------------------------------------------------------------------------------------------------------ |
| elasticsearch        | string | 目标集群的名称                                                                                               |
| scroll_time          | string | Scroll 回话超时时间                                                                                          |
| batch_size           | int    | Scroll 批次大小，默认 `5000`                                                                                 |
| slice_size           | int    | Slice 大小，默认 `1`                                                                                         |
| sort_type            | string | 文档排序类型，默认 `asc`                                                                                     |
| sort_field           | string | 文档排序字段                                                                                                 |
| indices              | string | 索引                                                                                                         |
| level                | string | 请求处理级别，可以设置为 `cluster` 则表示请求不进行节点和分片级别的拆分，适用于 Elasticsearch 前有代理的情况 |
| query                | string | 查询过滤条件                                                                                                 |
| fields               | string | 要返回的字段列表                                                                                             |
| sort_document_fields | bool   | hash 计算之前是否对 `_source` 里面的字段进行排序，默认 `false`                                               |
| hash_func            | string | hash 函数，可选 `xxhash32`、`xxhash64`、`fnv1a`，默认 `xxhash32`                                             |
| output_queue         | string | 输出结果的队列名称                                                                                           |
