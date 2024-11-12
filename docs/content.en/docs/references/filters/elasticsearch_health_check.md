---
title: "elasticsearch_health_check"
---

# elasticsearch_health_check

## Description

The elasticsearch_health_check filter is used to detect the health status of Elasticsearch in traffic control mode.
When a back-end fault occurs, the filter triggers an active cluster health check without waiting for the results of the default polling check of Elasticsearch. Traffic control can be configured to enable the filter to send check requests to the back-end Elasticsearch at a maximum of once per second.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: elasticsearch_health_check
    filter:
      - elasticsearch_health_check:
          elasticsearch: dev
```

## Parameter Description

| Name          | Type   | Description                                                                    |
| ------------- | ------ | ------------------------------------------------------------------------------ |
| elasticsearch | string | Cluster ID                                                                     |
| interval      | int    | Minimum interval for executing requests, in seconds. The default value is `1`. |
