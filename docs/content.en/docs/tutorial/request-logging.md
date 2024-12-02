---
title: "Elasticsearch Search Requests Analysis/Audit"
weight: 20
---

# Elasticsearch Search/Request Log Analysis/Audit

INFINI Gateway can track and record all requests that pass through the gateway and analyze requests sent to Elasticsearch, to figure out request performance and service running status.

{{% load-img "/img/dashboard-1.jpg" "" %}}

## Gateway configuration modification

After extracting the Extreme Gateway installation package, there will be a default configuration file called ‘gateway.yml’. With a simple modification, traffic analysis can be achieved. Typically, only this section needs to be modified. The configuration items after this will reference the defined content through variables.

```
env:
  LOGGING_ES_ENDPOINT: http://localhost:9200
  LOGGING_ES_USER: elastic
  LOGGING_ES_PASS: password
  PROD_ES_ENDPOINT: http://localhost:9200
  PROD_ES_USER: elastic
  PROD_ES_PASS: password
  GW_BINDING: "0.0.0.0:8000"
  API_BINDING: "0.0.0.0:2900"
```

The above configuration defines two ES clusters and the gateway’s listener information.

- LOGGING_ES_ENDPOINT : Define the access information of the log cluster, and all request logs will be written to this cluster.
- PROD_ES_ENDPOINT : Define the access information of the production cluster, the gateway will proxy this cluster.
- \*\_ES_USER and \*\_ES_PASS : Define the authentication information of the cluster.
- API_BINDING : Define the address and port that the gateway API service listens on.
- GW_BINDING Define the address and port that the gateway proxy service listens on.

In the test environment, the log cluster and production cluster can be the same.  
Please make sure that requests to access the ES cluster are sent to the address and port that the gateway proxy service is listening on.

The gateway comes with a cache function. If you need to enable this feature, please modify the default_flow configuration as follows:

```
  - name: default_flow
    filter:
      - get_cache:
      - elasticsearch:
          elasticsearch: prod
          max_connection_per_node: 1000
      - set_cache:
```

## INFINI Easysearch

`INFINI easysearch` supports [higher compression rates](https://www.infinilabs.com/blog/2023/easysearch-storage-compression/), which is more conducive to saving disk space.  
If the log cluster is using `INFINI easysearch`, it is important to install the index-management plugin.  
Please click [here](https://www.infinilabs.com/docs/latest/easysearch/getting-started/install/#%E6%8F%92%E4%BB%B6%E5%AE%89%E8%A3%85) to view the plugin installation documentation

```
bin/easysearch-plugin install index-management
```

After installing the plugin, restart for it to take effect.

## Configure the index template.

If you are already using `INFINI Console`, you can skip configuring the index lifecycle and index template because they have already been automatically created.

Execute the following command on the log cluster to create an index template for the log index.
{{% load-img "/img/create_template.png" "" %}}

> Please note that you may need to modify the above template settings before executing, for example, by adding the routing.allocation.require parameter to specify the node attribute where the index is created.

{{< expand "Click to expand the Elasticsearch’s template definition" "..." >}}

```
PUT _template/.infini_requests_logging-rollover
{
   "order": 100000,
   "index_patterns": [
       ".infini_requests_logging*"
   ],
   "settings": {
     "index": {
       "format": "7",
       "lifecycle": {
         "name" : "ilm_.infini_metrics-30days-retention",
         "rollover_alias" : ".infini_requests_logging"
       },
       "codec": "best_compression",
       "number_of_shards": "1",
       "translog": {
         "durability": "async"
       }
     }
   },
   "mappings": {
     "dynamic_templates": [
       {
         "strings": {
           "mapping": {
             "ignore_above": 256,
             "type": "keyword"
           },
           "match_mapping_type": "string"
         }
       }
     ],
     "properties": {
       "request": {
         "properties": {
           "body": {
             "type": "text"
           }
         }
       },
       "response": {
         "properties": {
           "body": {
             "type": "text"
           }
         }
       },
       "timestamp": {
         "type": "date"
       }
     }
   },
   "aliases": {}
 }


DELETE .infini_requests_logging-00001
PUT .infini_requests_logging-00001
{
  "settings": {
      "index.lifecycle.rollover_alias":".infini_requests_logging"
    , "refresh_interval": "5s"
  },
  "aliases":{
    ".infini_requests_logging":{
      "is_write_index":true
    }
  }
}

```

{{< /expand >}}
{{< expand "Click to expand INFINI Easysearch’s template definition for a 50% reduction in storage" "..." >}}

```
PUT _template/.infini_requests_logging-rollover
{
   "order": 100000,
   "index_patterns": [
       ".infini_requests_logging*"
   ],
   "settings": {
     "index": {
       "format": "7",
       "lifecycle": {
         "name" : "ilm_.infini_metrics-30days-retention",
         "rollover_alias" : ".infini_requests_logging"
       },
       "codec": "ZSTD",
       "source_reuse": true，
       "number_of_shards": "1",
       "translog": {
         "durability": "async"
       }
     }
   },
   "mappings": {
     "dynamic_templates": [
       {
         "strings": {
           "mapping": {
             "ignore_above": 256,
             "type": "keyword"
           },
           "match_mapping_type": "string"
         }
       }
     ],
     "properties": {
       "request": {
         "properties": {
           "body": {
             "type": "text"
           }
         }
       },
       "response": {
         "properties": {
           "body": {
             "type": "text"
           }
         }
       },
       "timestamp": {
         "type": "date"
       }
     }
   },
   "aliases": {}
 }


DELETE .infini_requests_logging-00001
PUT .infini_requests_logging-00001
{
  "settings": {
      "index.lifecycle.rollover_alias":".infini_requests_logging"
    , "refresh_interval": "5s"
  },
  "aliases":{
    ".infini_requests_logging":{
      "is_write_index":true
    }
  }
}

```

{{< /expand >}}

## Configuring the Index Lifecycle

{{< expand "Click to expand the definition of the index lifecycle" "..." >}}

```
PUT _ilm/policy/ilm_.infini_metrics-30days-retention
{
  "policy": {
    "phases": {
      "hot": {
        "min_age": "0ms",
        "actions": {
          "rollover": {
            "max_age": "30d",
            "max_size": "50gb"
          },
          "set_priority": {
            "priority": 100
          }
        }
      },
      "delete": {
        "min_age": "30d",
        "actions": {
          "delete": {
          }
        }
      }
    }
  }
}
```

{{< /expand >}}

## Importing the Dashboard

Download the latest dashboard [INFINI-Gateway-7.9.2-2021-01-15.ndjson.zip](https://pan.baidu.com/s/1iIXCrmMH-24fSzwcvIn8zg?pwd=gm2x#list/path=%2Fdashboard&parentPath=%2F) for Kibana 7.9 and import it into Kibana of the `dev` cluster as follows:

{{% load-img "/img/import-dashboard.jpg" "" %}}

## Starting the Gateway

Start the gateway.

```
➜ ./bin/gateway
   ___   _   _____  __  __    __  _
  / _ \ /_\ /__   \/__\/ / /\ \ \/_\ /\_/\
 / /_\///_\\  / /\/_\  \ \/  \/ //_\\\_ _/
/ /_\\/  _  \/ / //__   \  /\  /  _  \/ \
\____/\_/ \_/\/  \__/    \/  \/\_/ \_/\_/

[GATEWAY] A light-weight, powerful and high-performance elasticsearch gateway.
[GATEWAY] 1.0.0_SNAPSHOT, a17be4c, Wed Feb 3 00:12:02 2021 +0800, medcl, add extra retry for bulk_indexing
[02-03 13:51:35] [INF] [instance.go:24] workspace: data/gateway/nodes/0
[02-03 13:51:35] [INF] [api.go:255] api server listen at: http://0.0.0.0:2900
[02-03 13:51:35] [INF] [runner.go:59] pipeline: request_logging_index started with 1 instances
[02-03 13:51:35] [INF] [entry.go:267] entry [es_gateway] listen at: http://0.0.0.0:8000
[02-03 13:51:35] [INF] [app.go:297] gateway now started.
```

## Modifying Application Configuration

Replace the Elasticsearch address with the gateway address for applications directed to the Elasticsearch address (such as Beats, Logstash, and Kibana).
Assume that the gateway IP address is `192.168.3.98`. Modify the Kibana configuration as follows:

```
# The Kibana server's name.  This is used for display purposes.
#server.name: "your-hostname"

# The URLs of the Elasticsearch instances to use for all your queries.
elasticsearch.hosts: ["https://192.168.3.98:8000"]
elasticsearch.customHeaders: { "app": "kibana" }

# When this setting's value is true Kibana uses the hostname specified in the server.host
# setting. When the value of this setting is false, Kibana uses the hostname of the host
# that connects to this Kibana instance.
#elasticsearch.preserveHost: true

# Kibana uses an index in Elasticsearch to store saved searches, visualizations and
# dashboards. Kibana creates a new index if the index doesn't already exist.
#kibana.index: ".kibana"

# The default application to load.
#kibana.defaultAppId: "home"
```

Save the configuration and restart Kibana.

## Checking the Results

All requests that access Elasticsearch through the gateway can be monitored.

{{% load-img "/img/dashboard-1.jpg" "" %}}
{{% load-img "/img/dashboard-2.jpg" "" %}}
{{% load-img "/img/dashboard-3.jpg" "" %}}
