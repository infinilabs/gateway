---
title: "elasticsearch"
---

# elasticsearch

## Description

The elasticsearch filter is used to forward requests to back-end Elasticsearch clusters.

## Configuration Example

Before using the elasticsearch filter, define one Elasticsearch cluster configuration node as follows:

```
elasticsearch:
- name: prod
  enabled: true
  endpoint: http://192.168.3.201:9200
```

The following shows a flow configuration example.

```
flow:
  - name: cache_first
    filter:
      - elasticsearch:
          elasticsearch: prod
```

The preceding example forwards requests to the `prod` cluster.

## Automatic Update

For a large cluster that contains many nodes, it is almost impossible to configure all back-end nodes individually. Instead, you only need to enable auto-discovery of back-end nodes on the Elasticsearch module. See the following example.

```
elasticsearch:
- name: prod
  enabled: true
  endpoint: http://192.168.3.201:9200
  discovery:
    enabled: true
    refresh:
      enabled: true
  basic_auth:
    username: elastic
    password: pass
```

Then, enable automatic configuration refresh on the filter. Now, all back-end nodes can be accessed and the status of online and offline nodes is automatically updated. See the following example.

```
flow:
  - name: cache_first
    filter:
      - elasticsearch:
          elasticsearch: prod
          refresh:
            enabled: true
            interval: 30s
```

## Setting the Weight

If there are many back-end clusters, INFINI Gateway allows you to set different access weights for different nodes. See the following configuration example.

```
flow:
  - name: cache_first
    filter:
      - elasticsearch:
          elasticsearch: prod
          balancer: weight
          refresh:
            enabled: true
            interval: 30s
          weights:
            - host: 192.168.3.201:9200
              weight: 10
            - host: 192.168.3.202:9200
              weight: 20
            - host: 192.168.3.203:9200
              weight: 30
```

In the above example, the traffic destined for an Elasticsearch cluster is distributed to the `203`, `202`, and `201` nodes at a ratio of `3：2：1`.

## Filtering Node

INFINI Gateway can also filter requests based on node IP address, label, or role to avoid sending requests to specific nodes, such as the master and cold nodes. See the following configuration example.

```
flow:
  - name: cache_first
    filter:
      - elasticsearch:
          elasticsearch: prod
          balancer: weight
          refresh:
            enabled: true
            interval: 30s
          filter:
            hosts:
              exclude:
                - 192.168.3.201:9200
              include:
                - 192.168.3.202:9200
                - 192.168.3.203:9200
            tags:
              exclude:
                - temp: cold
              include:
                - disk: ssd
            roles:
              exclude:
                - master
              include:
                - data
                - ingest
```

## Parameter Description

| Name                     | Type     | Description                                                                                                                                                                                                                                                         |
| ------------------------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| elasticsearch            | string   | Name of an Elasticsearch cluster                                                                                                                                                                                                                                    |
| max_connection_per_node  | int      | Maximum number of TCP connections that are allowed to access each node of an Elasticsearch cluster. The default value is `5000`.                                                                                                                                    |
| max_response_size        | int      | Maximum size of the message body returned in response to an Elasticsearch request. The default value is `100*1024*1024`.                                                                                                                                            |
| max_conn_wait_timeout    | duration | Timeout duration for Elasticsearch to wait for an idle connection. The default value is `30s`.                                                                                                                                                                      |
| max_idle_conn_duration   | duration | Idle duration of an Elasticsearch connection. The default value is `30s`.                                                                                                                                                                                           |
| max_retry_times          | duration | Limit the number of retries on Elasticsearch errors, default `0`                                                                                                                                                                                                    |
| max_conn_duration        | duration | Duration of an Elasticsearch connection. The default value is `0s`.                                                                                                                                                                                                 |
| timeout                  | duration | Timeout duration to wait for the response. The default value is `30s`. Warning: `timeout` will not terminate the request, it will continue in the background. If response time is too long and the connection pool is full, try to set `read_timeout`.              |
| dial_timeout             | duration | Timeout duration to wait for dialing the remote host. The default value is `3s`.                                                                                                                                                                                    |
| read_timeout             | duration | Read timeout duration of an Elasticsearch request. The default value is `0s`.                                                                                                                                                                                       |
| write_timeout            | duration | Write timeout duration of an Elasticsearch request. The default value is `0s`.                                                                                                                                                                                      |
| read_buffer_size         | int      | Read cache size for an Elasticsearch request. The default value is `4096*4`.                                                                                                                                                                                        |
| write_buffer_size        | int      | Write cache size for an Elasticsearch request. The default value is `4096*4`.                                                                                                                                                                                       |
| tls_insecure_skip_verify | bool     | Whether to ignore TLS certificate verification of an Elasticsearch cluster. The default value is `true`.                                                                                                                                                            |
| max_retry_times                  | int      | The maximum number of retry attempts for requests.     The default value is `5`.                                                                                                                                 |
| retry_on_backend_failure         | bool     | Whether to retry requests when backend failures occur. Used to switch to another available host. The default value is `true`.                                                                           |
| retry_readonly_on_backend_failure| bool     | Whether to retry readonly requests (e.g., `GET`/`HEAD`) on backend failure. This is generally safe as it does not risk data duplication or corruption.   The default value is `true`.                                |
| retry_writes_on_backend_failure  | bool     | Whether to retry write operations (e.g., `POST`/`PUT`/`PATCH`) on backend failure. Use with caution, as retries can lead to duplicate writes. Recommended to use with additional filters. The default value is `false`. |
| retry_on_backend_busy            | bool     | Whether to retry requests when the backend is busy with status code `429`. This helps handle temporary overloads or throttling.    The default value is `false`.                                                                            |
| retry_delay_in_ms                | int      | The delay in milliseconds between retry attempts. Does not apply when switching hosts. The default value is `1000`.                                                               |
| balancer                 | string   | Load balancing algorithm of a back-end Elasticsearch node. Currently, only the `weight` weight-based algorithm is available.                                                                                                                                        |
| skip_metadata_enrich     | bool   | Whether to skip the processing of Elasticsearch metadata and not add `X-*` metadata to the header of the request and response                     |
| refresh.enable           | bool     | Whether to enable automatic refresh of node status changes, to perceive changes in the back-end Elasticsearch topology                                                                                                                                              |
| refresh.interval         | int      | Interval of the node status refresh                                                                                                                                                                                                                                 |
| weights                  | array    | Priority of a back-end node. A node with a larger weight is assigned a higher proportion of request forwarding.                                                                                                                                                     |
| filter                   | object   | Filtering rules for back-end Elasticsearch nodes. Rules can be set to forward requests to a specific node.                                                                                                                                                          |
| filter.hosts             | object   | Filtering based on the access address of Elasticsearch                                                                                                                                                                                                              |
| filter.tags              | object   | Filtering based on the label of Elasticsearch                                                                                                                                                                                                                       |
| filter.roles             | object   | Filtering based on the role of Elasticsearch                                                                                                                                                                                                                        |
| filter.\*.exclude        | array    | Conditions for excluding. Any matched node is denied handling requests as a proxy.                                                                                                                                                                                  |
| filter.\*.include        | array    | Elasticsearch nodes that meet conditions are allowed to handle requests as a proxy. When the exclude parameter is not configured but include is configured, any condition in include must be met. Otherwise, the node is not allowed to handle requests as a proxy. |
