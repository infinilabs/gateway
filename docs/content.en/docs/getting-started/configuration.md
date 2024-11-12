---
weight: 30
title: Configuring the Gateway
---

# Configuration

The configuration of INFINI Gateway can be modified in multiple ways.

## CLI Parameters

INFINI Gateway provides the following CLI parameters:

```
✗ ./bin/gateway --help
Usage of ./bin/gateway:
  -config string
        the location of config file, default: gateway.yml (default "gateway.yml")
  -debug
        run in debug mode, gateway will quit with panic error
  -log string
        the log level,options:trace,debug,info,warn,error (default "info")
  -v    version
```

The parameters are described as follows:

- config: Specifies the name of a configuration file. The default configuration file name is `gateway.yml` in the directory where the currently executed command is located. If your configuration file is stored elsewhere, you can specify the parameter to select it.
- daemon: Switches the gateway to the background. It needs to be used jointly with `pidfile` to save the process ID and facilitate subsequent process operations.

## Configuration File

Most of the configuration of INFINI Gateway can be completed using `gateway.yml`. After the configuration is modified, the gateway program needs to be restarted to make the configuration take effect.

### Defining an Entry

Each gateway must expose at least one service entrance to receive operation requests of services. In INFINI Gateway, the service entrance is called an `entry`, which can be defined using the following parameters:

```
entry:
  - name: es_gateway
    enabled: true
    router: default
    network:
      binding: 0.0.0.0:8000
```

The above configuration defines one service entry named `es_gateway`, the address listened to is `0.0.0.0:8000`, and one router named `default` is used to process requests.

### Defining a Router

INFINI Gateway judges the flow direction based on routers. A typical example of router configuration is as follows:

```
router:
  - name: default
    default_flow: cache_first
```

This example defines one router named `default`, which is also the main flow for service handling. Request forwarding, filtering, caching, and other operations are performed in this flow.

### Defining a Flow

One request flow defines a series of work units for request handling. It adopts a typical pipeline work mode. One typical configuration example is as follows:

```
flow:
  - name: cache_first
    filter:
      - get_cache:
      - elasticsearch:
          elasticsearch: prod
      - set_cache:
```

The configuration example defines a flow named `cache_first`, which uses three different filters: `get_cache`, `elasticsearch`, and `set_cache`. These filters are executed in their configuration sequence. Note that each filter name must be appended with one colon (`:`).
The processing results of the filters are as follows:

- get_cache: This filter is mainly used to get data from the cache. If the same request has been received before and data is cached in the cache, which is within the validity period, this filter can directly take and return the cached data immediately, without further processing.
- elasticsearch: This filter is used to forward requests to back-end Elasticsearch clusters and further transfer responses returned by Elasticsearch.
- set_cache: This filter caches execution results to the local memory. It has some parameter restrictions such as the status code and request size, and expiration time is set for the filter so that results in the cache can be used directly when the same request is received next time. It is generally used together with `get_cache`.

### Defining a Resource

Resources here refer to Elasticsearch back-end server resources. INFINI Gateway supports multiple Elasticsearch clusters. It can forward requests to different clusters and supports blue/green deployment and smooth evolution under canary deployment of requests. The following example shows how to define an Elasticsearch back-end resource.

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

The `endpoint` parameter is used to set the access address for Elasticsearch. If identity authentication is enabled for Elasticsearch, you can use `basic_auth` to specify the username and password and the user must have the permission to obtain cluster status information.
The `discover` parameter is used to enable automatic discovery to automatically detect the status of back-end nodes and identify new and offline nodes.

After these basic configurations have been completed, INFINI Gateway can normally handle Elasticsearch requests as a proxy. For details about parameters of each component, see the [Reference](../references/filters/).
