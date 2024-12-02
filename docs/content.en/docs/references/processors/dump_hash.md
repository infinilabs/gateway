---
title: "dump_hash"
deprecated: true
---

# dump_hash

## Description

The dump_hash processor is used to export index documents of a cluster and calculate the hash value.

## Configuration Example

A simple example is as follows:

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

## Parameter Description

| Name                 | Type   | Description                                                                                                                                                                                                                |
| -------------------- | ------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| elasticsearch        | string | Name of a target cluster                                                                                                                                                                                                   |
| scroll_time          | string | Scroll session timeout duration                                                                                                                                                                                            |
| batch_size           | int    | Scroll batch size, which is set to `5000` by default                                                                                                                                                                       |
| slice_size           | int    | Slice size, which is set to `1` by default                                                                                                                                                                                 |
| sort_type            | string | Document sorting type, which is set to `asc` by default                                                                                                                                                                    |
| sort_field           | string | Document sorting field                                                                                                                                                                                                     |
| indices              | string | Index                                                                                                                                                                                                                      |
| level                | string | Request processing level, which can be set to `cluster`, indicating that node- and shard-level splitting are not performed on requests. It is applicable to scenarios in which there is a proxy in front of Elasticsearch. |
| query                | string | Query filter conditions                                                                                                                                                                                                    |
| fields               | string | List of fields to be returned                                                                                                                                                                                              |
| sort_document_fields | bool   | Whether to sort fields in `_source` before the hash value is calculated. The default value is `false`.                                                                                                                     |
| hash_func            | string | Hash function, which can be set to `xxhash32`, `xxhash64`, or `fnv1a`. The default value is `xxhash32`.                                                                                                                    |
| output_queue         | string | Name of a queue that outputs results                                                                                                                                                                                       |
