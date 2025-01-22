---
title: "与 Elasticsearch-Hadoop 集成"
weight: 100
---

# 与 Elasticsearch-Hadoop 集成

Elasticsearch-Hadoop 默认会通过某个种子节点拿到后端的所有 Elasticsearch 节点，可能存在热点和请求分配不合理的情况，
为了提高后端 Elasticsearch 节点的资源利用率，可以通过极限网关来实现后端 Elasticsearch 节点访问的精准路由。

## 写入加速

如果是通过 Elasticsearch-Hadoop 来进行数据导入，可以通过修改 Elasticsearch-Hadoop 程序的以下参数来访问极限网关来提升写入吞吐，如下：

| 名称                   | 类型   | 说明                                                        |
| ---------------------- | ------ | ----------------------------------------------------------- |
| es.nodes               | string | 设置访问网关的地址列表，如：`localhost:8000,localhost:8001` |
| es.nodes.discovery     | bool   | 设置为 `false`，不采用 sniff 模式，只访问配置的后端节点列表 |
| es.nodes.wan.only      | bool   | 设置为 `true`，代理模式，强制走网关地址                     |
| es.batch.size.entries  | int    | 适当调大批次文档数，提升吞吐，如 `5000`                     |
| es.batch.size.bytes    | string | 适当调大批次传输大小，提升吞吐，如 `20mb`                   |
| es.batch.write.refresh | bool   | 设置为 `false`，避免主动刷新，提升吞吐                      |

## 相关链接

- [Elasticsearch-Hadoop 配置参数文档](https://www.elastic.co/guide/en/elasticsearch/hadoop/master/configuration.html)
