# INFINI Gateway

## Introduction

![](./docs/static/img/banner.jpg)

**INFINI Gateway** is a high performance gateway for Elasticsearch/OpenSearch/Easysearch. It offers a broad range of features and is easy to use. INFINI Gateway works in the same way as a common reverse proxy.
It is usually deployed in front of Elasticsearch/OpenSearch/Easysearch clusters. All requests are sent to the gateway instead of Elasticsearch/OpenSearch/Easysearch, and then the gateway forwards the requests to the back-end Elasticsearch/OpenSearch/Easysearch clusters.
The gateway is deployed between the client and Elasticsearch/OpenSearch/Easysearch. Therefore, the gateway can be configured to perform index-level traffic control and throttling, cache acceleration for common queries, query request audit, and dynamic modification of query results.

## Features

> The application-layer INFINI Gateway is especially designed for Elasticsearch/OpenSearch/Easysearch and offers powerful features.

- High availability: The gateway supports non-stop indexing and is capable of automatically processing faults occurring on Elasticsearch/OpenSearch/Easysearch, without affecting normal data ingestion.
- Write acceleration: The gateway can automatically merge independent index requests into a bulk request, thereby reducing back-end pressure and improving indexing efficiency.
- Query acceleration: Query cache can be configured on INFINI Gateway and Kibana dashboards can accelerate the query seamlessly and intelligently to fully enhance search experience.
- Seamless retry: The gateway automatically processes faults occurring on Elasticsearch/OpenSearch/Easysearch, and migrates and retries query requests.
- Traffic cloning: The gateway can replicate traffic to multiple different Elasticsearch/OpenSearch/Easysearch clusters and supports traffic migration through canary deployment.
- One-click rebuilding: The optimized high-speed index rebuilding and automatic processing of incremental data enable the gateway to seamlessly switch between old and new indexes.
- Secure transmission: The gateway supports the Transport Layer Security (TLS) and Hypertext Transfer Protocol Secure (HTTPS) protocols. It can automatically generate certification files and supports specified trust certification files.
- Precision routing: The gateway supports the load balancing mode using multiple algorithms, in which load routing strategies can be separately configured for indexing and query, providing great flexibility.
- Traffic control and throttling: Multiple traffic control and throttling rules can be configured to implement index-level traffic control and ensure the stability of back-end clusters.
- Concurrency control: The gateway can control cluster- and node-level concurrent TCP connections to ensure the stability of back-end clusters and nodes.
- No single point of failure (SPOF): The built-in virtual IP-based high availability solution supports dual-node hot standby and automatic failover to prevent SPOFs.
- Request observability: The gateway is equipped with the logging and indicator monitoring features to fully analyze Elasticsearch/OpenSearch/Easysearch requests.


To learn more about Gateway, please visit: https://docs.infinilabs.com/gateway/

## Community

Fell free to join the Discord server to discuss anything around this project:

[https://discord.gg/4tKTMkkvVX](https://discord.gg/4tKTMkkvVX)

## License

INFINI Gateway is a truly open-source project, licensed under the [GNU Affero General Public License v3.0](https://opensource.org/licenses/AGPL-3.0).
We also offer a commercially supported, enterprise-ready version of the software.
For more details, please refer to our [license information](./LICENSE).