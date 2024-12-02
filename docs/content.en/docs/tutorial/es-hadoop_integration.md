---
title: "Integrate with Elasticsearch-Hadoop"
weight: 100
---

# Integrate with Elasticsearch-Hadoop

Elasticsearch-Hadoop utilizes a seed node to access all back-end Elasticsearch nodes by default. The hotspots and requests may be improperly allocated.
To improve the resource utilization of back-end Elasticsearch nodes, you can implement precision routing for the access to Elasticsearch nodes through INFINI Gateway.

## Write Acceleration

If you import data by using Elasticsearch-Hadoop, you can modify the following parameters of Elasticsearch-Hadoop to access INFINI Gateway, so as to improve the write throughput:

| Name                   | Type   | Description                                                                                                        |
| ---------------------- | ------ | ------------------------------------------------------------------------------------------------------------------ |
| es.nodes               | string | List of addresses used to access the gateway, for example, `localhost:8000,localhost:8001`                         |
| es.nodes.discovery     | bool   | When it is set to `false`, the sniff mode is not adopted and only the configured back-end nodes are accessed.      |
| es.nodes.wan.only      | bool   | When it is set to `true`, it indicates the proxy mode, in which data is forcibly sent through the gateway address. |
| es.batch.size.entries  | int    | Batch document quantity. Set the parameter to a larger value to improve throughput, for example, `5000`.           |
| es.batch.size.bytes    | string | Batch transmission size. Set the parameter to a larger value to improve throughput, for example, `20mb`.           |
| es.batch.write.refresh | bool   | Set it to `false` to prevent active refresh and improve throughput.                                                |

## Related Link

- [Elasticsearch-Hadoop Configuration Parameter Document](https://www.elastic.co/guide/en/elasticsearch/hadoop/master/configuration.html)
