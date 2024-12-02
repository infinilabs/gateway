---
title: "index_diff"
deprecated: true
---

# index_diff

## Description

The index_diff processor is used to compare differences between two result sets.

## Configuration Example

A simple example is as follows:

```
pipeline:
- name: bulk_request_ingest
  auto_start: true
  keep_running: true
  processor:
    - index_diff:
        diff_queue: "diff_result"
        buffer_size: 1
        text_report: true #If data needs to be saved to Elasticsearch, disable the function and start the diff_result_ingest task of the pipeline.
        source_queue: 'source_docs'
        target_queue: 'target_docs'
```

## Parameter Description

| Name         | Type   | Description                                                    |
| ------------ | ------ | -------------------------------------------------------------- |
| source_queue | string | Name of source data                                            |
| target_queue | string | Name of target data                                            |
| diff_queue   | string | Queue that stores difference results                           |
| buffer_size  | int    | Memory buffer size                                             |
| keep_source  | bool   | Whether difference results contain document source information |
| text_report  | bool   | Whether to output results in text form                         |
