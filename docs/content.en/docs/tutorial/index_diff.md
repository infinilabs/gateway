---
title: "Document-Level Index Diff Between Two Elasticsearch Clusters"
weight: 30
asciinema: true
---

# Document-Level Index Diff Between Two Elasticsearch Clusters

INFINI Gateway is able to compare differences between two different indexes in the same or different clusters. In scenarios in which application dual writes, CCR, or other data replication solutions are used, differences can be periodically compared to ensure data consistency.

## Function Demonstration

{{< asciinema key="/index_diff" speed="2"  autoplay="1"  start-at="0" rows="30" preload="1" >}}

## How Is This Feature Configured?

### Setting a Target Cluster

Modify the `gateway.yml` configuration file by setting two cluster resources `source` and `target` and adding the following configuration:

```
elasticsearch:
  - name: source
    enabled: true
    endpoint: http://localhost:9200
    basic_auth:
      username: test
      password: testtest
  - name: target
    enabled: true
    endpoint: http://localhost:9201
    basic_auth: #used to discovery full cluster nodes, or check elasticsearch's health and versions
      username: test
      password: testtest
```

### Configuring a Contrast Task

Add a service pipeline to handle the index document pulling and contrast of two clusters as follows:

```
pipeline:
  - name: index_diff_service
    auto_start: true
    keep_running: true
    processor:
    - dag:
        parallel:
          - dump_hash: #dump es1's doc
              indices: "medcl-test"
              scroll_time: "10m"
              elasticsearch: "source"
              output_queue: "source_docs"
              batch_size: 10000
              slice_size: 5
          - dump_hash: #dump es2's doc
              indices: "medcl-test"
              scroll_time: "10m"
              batch_size: 10000
              slice_size: 5
              elasticsearch: "target"
              output_queue: "target_docs"
        end:
          - index_diff:
              diff_queue: "diff_result"
              buffer_size: 1
              text_report: true #If data needs to be saved to Elasticsearch, disable the function and start the diff_result_ingest task of the pipeline.
              source_queue: 'source_docs'
              target_queue: 'target_docs'
```

In the above configuration, `dump_hash` is concurrently used to pull the `medcl-a` index of the `source` cluster and fetch the `medcl-b` index of the `target` cluster, and output results to terminals in text form.

### Outputting Results to Elasticsearch

If there are many difference results, you can save them to the `Elasticsearch` cluster, set the `text_report` parameter of the above `index_diff` processing unit to `false`, and add the following configuration:

```
pipeline:
  - name: diff_result_ingest
    auto_start: true
    keep_running: true
    processor:
      - json_indexing:
          index_name: "diff_result"
          elasticsearch: "source"
          input_queue: "diff_result"
          idle_timeout_in_seconds: 1
          worker_size: 1
          bulk_size_in_mb: 10 #in MB
```

Finally, import the [dashboard](https://github.com/medcl/infini-gateway/releases/download/1.2.0/index-diff-report-v7.12.ndjson.zip) to Kibana to achieve the following effect:

{{% load-img "/img/index-diff-dashboard.jpg" "" %}}
