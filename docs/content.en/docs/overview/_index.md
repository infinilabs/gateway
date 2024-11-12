---
weight: 1
title: "Overview"
---

# Overview

## Introduction

**INFINI Gateway** is a High-performance data gateway for search scenarios. It offers a broad range of features and is easy to use. INFINI Gateway works in the same way as a common reverse proxy.
It is usually deployed in front of Elasticsearch clusters. All requests are sent to the gateway instead of Elasticsearch, and then the gateway forwards the requests to the back-end Elasticsearch clusters.
The gateway is deployed between the client and Elasticsearch. Therefore, the gateway can be configured to perform index-level traffic control and throttling, cache acceleration for common queries, query request audit, and dynamic modification of query results.

## Features

INFINI Gateway caters to Elasticsearch well. Many Elasticsearch-related service scenarios and characteristics are taken into account in the design. Therefore, INFINI Gateway is tailored to provide many useful features for Elasticsearch.

{{< columns >}}

### Lightweight

INFINI Gateway is written in Golang. The installation package is only about 10 MB and has no external environment dependency. Deployment and installation are very simple. Users can simply download the binary executable file of the gateway program from the platform and execute the program file.

<--->

### Optimal Performance

INFINI Gateway is designed to run in an optimized state during programming. Test results show that INFINI Gateway provides a speed that is over 25% faster than mainstream gateway counterparts, and which has been optimized to allow Elasticsearch to double the write and query speeds.
{{< /columns >}}

{{< columns >}}

### Cross-Version Support

INFINI Gateway is compatible with different Elasticsearch versions to ensure seamless adaptation of the service code. The back-end Elasticsearch cluster versions can be upgraded seamlessly, which reduces the complexity of version upgrade and data migration.

<--->

### Observability

INFINI Gateway can dynamically intercept and analyze requests generated during the running of Elasticsearch. Users can learn about the running status of the entire cluster from indicators and logs, in an effort to improve performance and optimize services. It can be also used for auditing and slow query analysis. {{< /columns >}}

{{< columns >}}

### High Availability

INFINI Gateway has multiple built-in high availability (HA) solutions. The front-end request entry supports virtual IP-based dual-node hot standby. The back-end cluster supports auto perception of the cluster topology, auto discovery of nodes that go online/offline, auto processing of back-end faults, and auto retry and migration of requests.

<--->

### Flexible and Extensible

Each module of INFINI Gateway can be independently extended and each request can be flexibly handled and routed. INFINI Gateway supports intelligent learning of routes and provides rich internal filters. The processing logic of each request can be dynamically modified. INFINI Gateway can also be extended using plug-ins.

{{< /columns >}}

## Seamless Integration

The external interfaces provided by INFINI Gateway are fully compatible with Elasticsearch's native interfaces. The integration is very simple and can be completed by changing the configuration pointed to Elasticsearch to the address of the gateway.

{{% load-img "/img/oveview-diagram.jpg" "" %}}

## Why Is INFINI Gateway Needed?

I largely understand the above integration interaction diagram and am familiar with the use of Elasticsearch. Why do I need to place a gateway in front of it?

If the scale of your Elasticsearch cluster is quite large, consider the following scenarios:

### WAF and Security

The prevalence of Elasticsearch makes it a prime target for hackers, which necessitates the use of a Web application firewall (WAF).
Whether it's the use of cross-site script attacks, cross-site scripting injection, weak passwords, brute force cracking, or unreasonable query parameter abuse by programmers, INFINI Gateway is able to detect and verify requests from different Web application clients.
It utilizes a series of Elasticsearch security policies to ensure security and legitimacy and block illegitimate requests in real time.

### Cluster Upgrade

The Elasticsearch iteration is pretty fast and cluster upgrade needs to be handled frequently. However, the following points must be considered in the cluster upgrade:

- Minimal downtime: The service data writing and query cannot be interrupted due to the cluster upgrade. Data can be continuously written but data cannot be lost due to the restart of back-end nodes.
- Cluster traffic switching. You need to determine when and how to switch traffic from the old cluster to the new one, whether to modify the service code or configuration file, how to roll back and restore the system, and whether to release a new deployment package.

With INFINI Gateway, you do not need to care about the back-end Elasticsearch clusters in the service code but only need to access the fixed address of the gateway. Then, INFINI Gateway will solve all problems for you.

### Index Rebuilding

Index rebuilding is required when mappings or the tokenizer dictionary is changed. Data writing cannot be stopped during rebuilding, data must be consistent after rebuilding, and new data and modified data must be handled, which are cumbersome.
INFINI Gateway supports one-click index rebuilding and automatically records any document modifications that take place during rebuilding. It switches from the old index to the new one seamlessly after rebuilding, which is completely imperceptive to front-end applications.

### Throttling and Traffic Control

A cluster may break down due to burst traffic or become overburdened due to large indexes. For this, you need to manage abnormal traffic in order to protect the entire Elasticsearch cluster against abnormal traffic and even malicious attacks.
INFINI Gateway can control the traffic flexibly and allows setting index-level traffic control rules. There are a thousand traffic control rules for a thousand indexes.

### Slow Query

The built-in cache function of INFINI Gateway can cache the most common queries and warm up specific queries according to periodic query plans to ensure that queries are hit each time for front-end services, thereby increasing query speed and improving user's service query experience.

### Slow Indexing

INFINI Gateway can combine many small batches of Elasticsearch index requests from different clients into one large bulk request, and deliver the index requests to a specified node of a specified shard through shard-level precision routing.
In this way, back-end Elasticsearch does not need to forward the requests, saving Elasticsearch resources and bandwidth, and improving overall throughput and performance of the cluster.

### Request Mutation

What happens if you discover errors in query statements after the code goes live? With INFINI gateway, you do not need to worry about it. You can rewrite a specified query of a specified service online and correct the query statements dynamically, without re-publishing the application, which offers convenience and flexibility.
If you are not satisfied with the JSON query results returned by Elasticsearch, you can utilize INFINI gateway to dynamically replace query results, and even merge data from other sources such as Hbase and MySQL into the required JSON data, which is then returned to the client.

### Request Analysis

People complain about the slow response of Elasticsearch, but do you know which indexes in Elasticsearch are slow? Which queries cause slow response? Which users are accountable for the slow response?
INFINI Gateway tracks Elasticsearch from clusters to indexes, from indexes to queries, and from applications to users so that you know every detail about the Elasticsearch clusters.

In a word, using Elasticsearch together with INFINI gateway will give you a splendid experience.

## Architecture

The architecture diagram below shows the core modules of INFINI Gateway.

{{% load-img "/img/architecture.jpg" "" %}}

The modules that carry external requests as a proxy are as follows: entry, router, flow, and filter. One entry needs one router, one router can route requests to multiple flows, and one flow is composed of multiple filters.

### Entry

The entry module defines the request entry for the gateway. INFINI Gateway supports the Hypertext Transfer Protocol (HTTP) and Hypertext Transfer Protocol Secure (HTTPS) modes. It automatically generates certification files in HTTPS mode.

### Router

The router module mainly defines routing rules for requests, and routes requests to a specified flow according to the method and request address.

### Flow

The flow module mainly defines the processing logic of data. Each request will go through a series of filter operations, and a flow is used to organize these filter operations.

### Filter

The filter module is composed of several different filter components. Each filter is designed to cope with only one task, and multiple filters compose a single flow.

### Pipeline

The pipeline module is composed of several different processor components. Compared with a flow, a pipeline focuses on the processing of offline tasks.

### Queue

The queue module is an abstract message queue, such as local disk-based reliability message persistence, Redis, Kafka, and other adapters. Different back-end adapters can be set for queues based on scenarios.

At the bottom layer of the framework used by INFINI Gateway, there are some common modules, such as the API used to provide an external programming entry and the Elastic module used to handle the API encapsulation for different versions of Elasticsearch.

## Next

- View [Downloading and Installation](../getting-started/install/)
