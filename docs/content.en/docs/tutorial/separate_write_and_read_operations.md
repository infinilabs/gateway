---
title: "Gateway-based Read/Write Separation Setting"
weight: 30
draft: true
---

# Gateway-based Read/Write Separation Setting

When a query request is received, Elasticsearch randomly selects one data shard (primary shard or replica) for searching by default.
Some scenarios may require the separation of read and write requests, and generally different target nodes may need to be manually selected on the client, which is extremely cumbersome. A more convenient solution is to use a gateway.

## Based on the `preference` Parameter

Elasticsearch provides one useful `preference` parameter, which can control the priority of requested target resources and support the following parameters:
| Name | Description |
| --------- | ------------------------------------------------------------ |
| _only_local | Only accesses existing shard data on the local node. |
| \_local | Accesses shard data on the local node preferentially if the local node has relevant shard data. Otherwise, requests are forwarded to relevant nodes. |
| \_only_nodes:<node-id>,<node-id> | Only accesses shard data on a specific node. When shard data does not exist on a specific node, requests are forwarded to relevant nodes. |
| \_prefer_nodes:<node-id>,<node-id> | Accesses shard data on a specified node preferentially. |
| \_shards:<shard>,<shard> | Accesses data on a shard of a specific number. |
| <custom-string>| Any user-defined string (not starting with `_`). Requests that contain the same value of the string are used to access the shards in the same order. |
