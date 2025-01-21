---
title: "查询请求流量日志分析"
weight: 20
---

# 查询请求流量日志分析

极限网关能够跟踪记录经过网关的所有请求，可用来分析发送给 Elasticsearch 的请求情况，用于分析请求性能和了解业务运行情况。

{{% load-img "/img/dashboard-1.jpg" "" %}}

## 网关配置修改

极限网关安装包解压后，会有一个默认配置`gateway.yml`。只需对其进行简单的修改，就可实现流量分析目的。
通常我们只需修改此部分内容。后面的配置项会通过变量方式引用在此定义的内容。

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

上面的配置定义了两个 ES 集群和网关的监听信息。

- LOGGING_ES_ENDPOINT 定义日志集群的访问信息，所有请求记录将写入该集群。
- PROD_ES_ENDPOINT 定义生产集群的访问信息，网关将代理此集群。
- \*\_ES_USER 和\*\_ES_PASS 定义集群的认证信息。
- API_BINDING 定义网关 API 服务监听的地址和端口。
- GW_BINDING 定义网关代理服务监听的地址和端口。

在测试环境中，日志集群和生产集群可以是同一个。  
请确保将访问 ES 集群的请求发往网关代理服务监听的地址和端口。

网关自带`cache`功能，如果需要启用该功能，请修改`default_flow`配置如下

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

`INFINI easysearch`支持更高的[压缩率](https://www.infinilabs.com/blog/2023/easysearch-storage-compression/)，更利于节省磁盘空间。  
如果`logging`集群使用的是`INFINI easysearch`，注意要安装`index-management`插件。  
[点此查看插件安装文档](https://www.infinilabs.com/docs/latest/easysearch/getting-started/install/#%E6%8F%92%E4%BB%B6%E5%AE%89%E8%A3%85)

```
bin/easysearch-plugin install index-management
```

插件安装完后重启生效。

## 配置索引模板

如果你已经在使用[INFINI Console](https://www.infinilabs.com/docs/latest/console/getting-started/install/)了，可跳过配置索引生命周期和索引模板，因为这些都已经自动建好了。

在 `logging` 集群上执行下面的命令创建日志索引的模板。

{{% load-img "/img/create_template.png" "" %}}

> 请注意，您可能需要在执行之前修改上面的模板设置，例如增加 `routing.allocation.require` 参数，指定索引创建时存放的节点属性。

{{< expand "展开查看 Elasticsearch 的模板定义" "..." >}}

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
{{< expand "展开查看 INFINI Easysearch 的模板定义 存储减50%" "..." >}}

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

## 配置索引生命周期

{{< expand "展开查看索引生命周期的定义" "..." >}}

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

## 导入仪表板

下载面向 Kibana 7.9 的最新的仪表板 [INFINI-Gateway-7.9.2-2021-01-15.ndjson.zip](https://pan.baidu.com/s/1iIXCrmMH-24fSzwcvIn8zg?pwd=gm2x#list/path=%2Fdashboard&parentPath=%2F)，在 `dev` 集群的 Kibana 里面导入，如下：

{{% load-img "/img/import-dashboard.jpg" "" %}}

## 启动网关

接下来，就可以启动网关，。

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

## 修改应用配置

将之前指向 Elasticsearch 地址的应用（如 Beats、Logstash、Kibana 等）换成网关的地址。
假设网关 IP 是 `192.168.3.98`，则修改 Kibana 配置如下：

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

保存配置并重启 Kibana。

## 查看效果

现在任何通过网关访问 Elasticsearch 的请求都能被监控到了。

{{% load-img "/img/dashboard-1.jpg" "" %}}
{{% load-img "/img/dashboard-2.jpg" "" %}}
{{% load-img "/img/dashboard-3.jpg" "" %}}
