--- 
title: INFINI Gateway 
type: docs 
---

# INFINI Gateway

## Introduction

**INFINI Gateway** is a high performance gateway for Elasticsearch. It offers a broad range of features and is easy to use. INFINI Gateway works in the same way as a common reverse proxy. 
It is usually deployed in front of Elasticsearch clusters. All requests are sent to the gateway instead of Elasticsearch, and then the gateway forwards the requests to the back-end Elasticsearch clusters. 
The gateway is deployed between the client and Elasticsearch. Therefore, the gateway can be configured to perform index-level traffic control and throttling, cache acceleration for common queries, query request audit, and dynamic modification of query results.

{{< button relref="./docs/overview/" >}}Learn More{{< /button >}}


## Features

> The application-layer INFINI Gateway is especially designed for Elasticsearch and offers powerful features.

- High availability: The gateway supports non-stop indexing and is capable of automatically processing faults occurring on Elasticsearch, without affecting normal data ingestion.
- Write acceleration: The gateway can automatically merge independent index requests into a bulk request, thereby reducing back-end pressure and improving indexing efficiency.
- Query acceleration: Query cache can be configured on INFINI Gateway and Kibana dashboards can accelerate the query seamlessly and intelligently to fully enhance search experience.
- Seamless retry: The gateway automatically processes faults occurring on Elasticsearch, and migrates and retries query requests.
- Traffic cloning: The gateway can replicate traffic to multiple different Elasticsearch clusters and supports traffic migration through canary deployment.
- One-click rebuilding: The optimized high-speed index rebuilding and automatic processing of incremental data enable the gateway to seamlessly switch between old and new indexes.
- Secure transmission: The gateway supports the Transport Layer Security (TLS) and Hypertext Transfer Protocol Secure (HTTPS) protocols. It can automatically generate certification files and supports specified trust certification files.
- Precision routing: The gateway supports the load balancing mode using multiple algorithms, in which load routing strategies can be separately configured for indexing and query, providing great flexibility.
- Traffic control and throttling: Multiple traffic control and throttling rules can be configured to implement index-level traffic control and ensure the stability of back-end clusters.
- Concurrency control: The gateway can control cluster- and node-level concurrent TCP connections to ensure the stability of back-end clusters and nodes.
- No single point of failure (SPOF): The built-in virtual IP-based high availability solution supports dual-node hot standby and automatic failover to prevent SPOFs.
- Request observability: The gateway is equipped with the logging and indicator monitoring features to fully analyze Elasticsearch requests.


{{< button relref="./docs/getting-started/install" >}}Get Started Now{{< /button >}}

## Community
Fell free to join the Discord server to discuss anything around this project:

https://discord.gg/4tKTMkkvVX


## Who Is Using INFINI Gateway?

If you are using INFINI Gateway and feel it pretty good, please [let us know](mailto:hello@infini.ltd). All our user cases are located [here](./docs/user-cases/). Thank you for your support.