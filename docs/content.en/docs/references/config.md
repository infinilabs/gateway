---
title: "Other Configurations"
weight: 250
---

# Other Configurations

## Advanced Usage

### Templates

Example:

```
configs.template:
  - name: "es_gw1"
    path: ./sample-configs/config_template.tpl
    variable:
      name: "es_gw1"
      binding_host: "0.0.0.0:8000"
      tls_on_entry: true
      elasticsearch_endpoint: "http://localhost:9200"
```

| Name                        | Type   | Description                                                |
| --------------------------- | ------ | ---------------------------------------------------------- |
| configs.template            | array  | Configuration templates, can specify multiple templates with corresponding parameters |
| configs.template[].name     | string | Name of the configuration                                    |
| configs.template[].path     | string | Template configuration path                                 |
| configs.template[].variable | map    | Template parameter settings, variables in the template are used as `$[[variable_name]]` |



### Environment Variables

The Gateway supports the use of environment variables for flexible parameter control within the configuration.

First, define the default values for environment variables in the configuration, as follows:

```
env:
  PROD_ES_ENDPOINT: http://localhost:9200
  PROD_ES_USER: elastic
  PROD_ES_PASS: password
```

Then, you can use environment variables in the configuration using the following syntax:

```
elasticsearch:
  - name: prod
    enabled: true
    endpoints:
      - $[[env.PROD_ES_ENDPOINT]]
    discovery:
      enabled: false
    basic_auth:
      username: $[[env.PROD_ES_USER]]
      password: $[[env.PROD_ES_PASS]]
```

Note that external environment variables take precedence over internal environment variable settings in the configuration. For example, to override environment variables when starting the program, use the following command:

```
PROD_ES_ENDPOINT=http://1.1.1.1:9200 LOGGING_ES_ENDPOINT=http://2.2.2.2:9201 ./bin/gateway
```

## Path

The instance configuration, data, and log directories.

Example:

```yaml
path.data: data
path.logs: log
path.configs: "config"
```

| Name                    | Type   | Description                                                                                                                                            |
| ----------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| path.data               | string | Data directory, default is `data`.                                                                                                                         |
| path.logs               | string | Log directory, default is `log`.                                                                                                                          |
| path.configs            | string | Configuration directory, default is `config`.                                                                                                                       |

## Log

The configuration for instance logs.

Example:

```yaml
log:
  level: info
  debug: false
```

| Name                    | Type   | Description                                                                                                                                            |
| ----------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| log.level               | string | Log level, default is `info`.                                                                                                                         |
| log.debug               | bool   | Whether to enable debug mode. When enabled, the program exits immediately in case of an exception, printing the complete stack trace. Used for debugging and fault localization. Default is `false`, do not enable in production as it may result in data loss.      |
| log.format              | bool   | Log format, default is `[%Date(01-02) %Time] [%LEV] [%File:%Line] %Msg%n`. [Format References](https://github.com/cihub/seelog/wiki/Format-reference). |
| log.disable_file_output | bool   | Whether to disable local file log output, default is `false`. Use this in container environments if you don't want local log output.                                                          |

## Configs

Manage the configuration of instance.

Example:

```yaml
configs:
  auto_reload: true
  managed: true
  panic_on_config_error: false 
  interval: "1s"
  servers:
    - "http://localhost:9000"
  max_backup_files: 5
  soft_delete: false
  tls:
    enabled: false
    cert_file: /etc/ssl.crt
    key_file: /etc/ssl.key
    skip_insecure_verify: false
```

| Name | Type | Description |
| --- | --- | --- |
| configs.auto_reload | bool | Whether it supports dynamically loading configuration files under the path.configs path. |
| configs.managed | bool | Whether configuration management by the configuration center is supported. |
| configs.servers | []string | Configuration center address |
| configs.interval | string | Configuration synchronization interval |
| configs.soft_delete | bool | The deletion of configuration files is soft deletion, default is `true`. |
| configs.panic_on_config_error | bool | If there is an error in configuration loading, it will crash directly, default `true` |
| configs.max_backup_files | int | The maximum number of configuration file backups, default `10`. |
| configs.valid_config_extensions | []string | Valid configuration file suffixes, default `.tpl`, `.json`, `.yml`, `.yaml` |
| configs.tls | object | TLS Configuration (Please refer to [TLS](#tls-配置)) |
| configs.always_register_after_restart | bool | Whether to register after the instance is restarted. When the instance runs in the K8S environment, this parameter needs to be enabled. |
| configs.allow_generated_metrics_tasks | bool | Allow automatic generation of collection metrics tasks. |
| configs.ignored_path | []string | Paths of configuration files that need to be ignored. |   

## Local Disk Queue

Example:

```
disk_queue:
  upload_to_s3: true
  s3:
    server: my_blob_store
    location: cn-beijing-001
    bucket: infini-store
  max_bytes_per_file: 102400
```

| Name                                        | Type   | Description                                                 |
| ------------------------------------------- | ------ | ------------------------------------------------------------ |
| disk_queue.min_msg_size                     | int    | Minimum byte limit for a single message sent to the queue, default is `1` |
| disk_queue.max_msg_size                     | int    | Maximum byte limit for a single message sent to the queue, default is `104857600` (100MB) |
| disk_queue.sync_every_records               | int    | Synchronization interval in terms of the number of records, default is `1000` |
| disk_queue.sync_timeout_in_ms               | int    | Synchronization interval in milliseconds, default is `1000` milliseconds |
| disk_queue.max_bytes_per_file               | int    | Maximum size of a single file in the local disk queue. If exceeded, a new file is created. Default is `104857600` (100MB) |
| disk_queue.max_used_bytes                   | int    | Maximum allowed storage space used by the local disk queue |
| disk_queue.warning_free_bytes               | int    | Free storage space threshold for disk space warnings, default is `10737418240` (10GB) |
| disk_queue.reserved_free_bytes              | int    | Protected value for free storage space on disk. Once reached, the disk becomes read-only and no more writes are allowed. Default is `5368709120` (5GB) |
| disk_queue.auto_skip_corrupted_file          | bool   | Whether to automatically skip corrupted disk files, default is `true` |
| disk_queue.upload_to_s3                     | bool   | Whether to upload disk queue files to S3, default is `false` |
| disk_queue.s3.async                         | bool   | Whether to asynchronously upload to the S3 server |
| disk_queue.s3.server                        | string | S3 server ID |
| disk_queue.s3.location                      | string | S3 server location |
| disk_queue.s3.bucket                        | string | S3 server bucket |
| disk_queue.retention.max_num_of_local_files | int    | Maximum number of files to retain locally after uploading to S3, default is `3` |
| disk_queue.compress.segment.enabled         | bool   | Whether to enable file-level compression, default is `false` |

## S3

Example:

```
s3:
  my_blob_store:
    endpoint: "192.168.3.188:9000"
    access_key: "admin"
    access_secret: "gogoaminio"
```

| Name                         | Type   | Description                  |
| ---------------------------- | ------ | ----------------------------- |
| s3.[id].endpoint             | string | S3 server address             |
| s3.[id].access_key           | string | S3 server key                 |
| s3.[id].access_secret        | string | S3 server secret key          |
| s3.[id].token                | string | S3 server token information   |
| s3.[id].ssl                  | bool   | Whether S3 server uses TLS   |
| s3.[id].skip_insecure_verify | bool   | Whether to skip TLS certificate verification for S3 server |

## Kafka

The Gateway supports using distributed Kafka as a backend queue. The related parameters are as follows.

| Name                            | Type   | Description                                                  |
| ------------------------------- | ------ | ------------------------------------------------------------- |
| kafka.enabled                   | bool   | Whether the Kafka module is enabled                          |
| kafka.default                   | bool   | Whether the Kafka module is the default queue implementation |
| kafka.num_of_partition          | int    | Default number of partitions, default is `1`                 |
| kafka.num_of_replica            | int    | Default number of replicas, default is `1`                   |
| kafka.producer_batch_max_bytes  | int    | Maximum size of the batch to submit, default is `50 * 1024 * 1024` |
| kafka.max_buffered_records      | int    | Maximum number of buffered request records, default is `10000` |
| kafka.manual_flushing           | bool   | Whether to enable manual flushing, default is `false`         |
| kafka.brokers                   | []string | Server address information                                    |
| kafka.username                  | string | User information                                             |
| kafka.password                  | string | Password information                                         |

## Badger

Badger is a lightweight disk-based KeyValue storage engine used by the Gateway to implement the KV module.

| Name                            | Type   | Description                                               |
| ------------------------------- | ------ | ---------------------------------------------------------- |
| badger.enabled                   | bool   | Whether to enable the KV module implemented by Badger, default is `true` |
| badger.single_bucket_mode        | bool   | Whether Badger module uses single bucket mode, default is `true` |
| badger.sync_writes               | bool   | Whether Badger module uses synchronous writes, default is `false` |
| badger.mem_table_size            | int64  | Size of the in-memory table used by Badger module, default is `10 * 1024 * 1024` (10485760) |
| badger.value_log_file_size       | int64  | Size of Badger module's log files, default is `1<<30 - 1` (1GB) |
| badger.value_log_max_entries     | int64  | Maximum number of log entries for Badger module, default is `1000000` |
| badger.value_threshold           | int64  | Value threshold for Badger module's log files, default is `1048576` (1MB) |
| badger.num_mem_tables            | int64  | Number of in-memory tables for Badger module, default is `1` |
| badger.num_level0_tables         | int64  | Number of Level0 in-memory tables for Badger module, default is `1` |

## Resource Limitations

| Name                       | Type   | Description                                                         |
| -------------------------- | ------ | ------------------------------------------------------------------- |
| resource_limit.cpu.max_num_of_cpus | int    | Maximum number of CPU cores allowed to be used, Linux only with `taskset` command available.             |
| resource_limit.cpu.affinity_list   | string | CPU affinity settings, e.g., `0,2,5` or `0-8`, Linux only with `taskset` command available.             |
| resource_limit.memory.max_in_bytes   | string | the max size of Memory to use, soft limit only           |


## Network Configuration

Common network configurations.

| Name                       | Type   | Description                                                         |
| -------------------------- | ------ | ------------------------------------------------------------------- |
| *.network.host               | string | Network address listened to by the service, for example, `192.168.3.10`              |
| *.network.port               | int    | Port address listened to by the service, for example, `8000`                         |
| *.network.binding            | string | Network binding address listened to by the service, for example, `0.0.0.0:8000`      |
| *.network.publish            | string | External access address listened to by the service, for example, `192.168.3.10:8000` |
| *.network.reuse_port         | bool   | Whether to reuse the network port for multi-process port sharing                     |
| *.network.skip_occupied_port | bool   | Whether to automatically skip occupied ports                                         |


## TLS Configuration

Example:

```
web:
  enabled: true
  embedding_api: true
  network:
    binding: $[[env.SERV_BINDING]]
  tls:
    enabled: false
    skip_insecure_verify: true
    default_domain: "api.coco.rs"
    auto_issue:
      enabled: true
      email: "hello@infinilabs.com"
      include_default_domain: true
      domains:
        - "www.coco.rs"
      provider:
        tencent_dns:
          secret_id: $[[keystore.TENCENT_DNS_ID]] #./bin/coco keystore add TENCENT_DNS_ID
          secret_key: $[[keystore.TENCENT_DNS_KEY]] #./bin/coco keystore add TENCENT_DNS_KEY
```

Common TLS configurations.

| Name                       | Type   | Description                                                         |
| -------------------------- | ------ | ------------------------------------------------------------------- |
| *.tls.enabled                | bool   | Whether TLS secure transmission is enabled or not, can auto generate cert files if not specified any cert files                                          |
| *.tls.ca_file              | string | Path to the public CA cert of the TLS security certificate                               |
| *.tls.cert_file              | string | Path to the public key of the TLS security certificate                               |
| *.tls.key_file               | string | Path to the private key of the TLS security certificate                              |
| *.tls.skip_insecure_verify   | bool   | Whether to ignore TLS certificate verification                                       |
| *.tls.default_domain   | string   | The default domain for auto generated certs                                    |
| *.tls.skip_domain_verify   | bool   | Whether to skip domain verify or not                                |
| *.tls.client_session_cache_size   | int   | Set the max cache of ClientSessionState entries for TLS session resumption   |


### Auto-Issue TLS Certificates

Both the `api` and `web` modules support auto-issuing TLS certificates via Let's Encrypt. This feature can be configured under `*.tls.auto_issue`:

| Name                                | Type    | Description                                                                                       |
|-------------------------------------|---------|---------------------------------------------------------------------------------------------------|
| *.tls.auto_issue.enabled        | bool    | Enables automatic issuance of TLS certificates using Let's Encrypt.                               |
| *.tls.auto_issue.path             | string  | Directory path where auto-issued certificates should be stored.                                   |
| *.tls.auto_issue.email            | string  | Contact email for certificate issuance notifications and expiry warnings.                         |
| *.tls.auto_issue.include_default_domain | bool | Whether to include the `default_domain` in the list of domains for auto-issuance.                 |
| *.tls.auto_issue.domains          | []string | List of additional domains for which TLS certificates will be issued.                             |
| *.tls.auto_issue.provider         | object  | Specifies the DNS provider configuration for DNS-based domain validation.                         |

#### DNS Provider Configuration (Tencent Cloud)

To support DNS-based verification with Tencent Cloud, configure the following within `*.tls.auto_issue.provider`:

| Name                     | Type    | Description                                                                                       |
|--------------------------|---------|---------------------------------------------------------------------------------------------------|
| `tencent_dns.secret_id`  | string  | Secret ID for Tencent Cloud API access.                                                           |
| `tencent_dns.secret_key` | string  | Secret Key for Tencent Cloud API access.                                                          |

To set up and store the Tencent Cloud credentials securely, use the keystore commands:
```bash
./bin/coco keystore add TENCENT_DNS_ID
./bin/coco keystore add TENCENT_DNS_KEY
```


## API

| Name                       | Type   | Description                                                         |
| -------------------------- | ------ | ------------------------------------------------------------------- |
| api.enabled | bool    | Whether to enable the API module, default is `true`           |
| api.network | object    | Networking config, please refer to common network configuration section         |
| api.tls | object    | TLS config, please refer to common TLS configuration section        |
| api.security | object    | Security config for API module     |
| api.security.enabled | bool    | Whether security is enabled or not     |
| api.security.username | string    | The username for security    |
| api.security.password | string    | The password for security   |
| api.cors.allowed_origins | []string    | The list of origins a cross-domain request can be executed from   |
| api.websocket | object    | Websocket config for API module |
| api.websocket.enabled | object    | Whether websocket is enabled or not |
| api.websocket.permitted_hosts | []string    | The list of hosts that permitted to access the websocket service |
| api.websocket.skip_host_verify | bool    | Whether websocket skip verify the host or not |

## Metrics

Configure collection of system metrics.

Example:

```yaml
metrics:
  enabled: true
  queue: metrics
  network:
    enabled: true
    summary: true
    details: true
  memory:
    metrics:
      - swap
      - memory
  disk:
    metrics:
      - iops
      - usage
  cpu:
    metrics:
      - idle
      - system
      - user
      - iowait
      - load
```

| Name | Type | Description |
| --- | --- | --- |
| enabled | bool | Whether to enable system metrics collection, default `true`. |
| queue | string | The queue name of metrics collection. |
| network | object | The Configuration of network metrics collection. |
| network.enabled | bool | Whether to enable network metrics collection, default `true`. |
| network.summary | bool | Whether to collect network summary metircs. |
| network.sockets | bool | Whether to collect network socket metircs. |
| network.throughput | bool | Whether to collect network throughput metircs. |
| network.details | bool | Whether to accumulate network IO metrics. |
| network.interfaces | []string | Specify the network interfaces to be collected, and all interfaces are default. |
| memory | object | The Configuration of memory metrics collection. |
| memory.enabled | bool | Whether to enable memory metrics collection, default `true` |
| memory.metrics | []string | Specified collection metrics, optional `swap`，`memory` |
| disk | object | The Configuration of disk metrics collection. |
| disk.metrics | []string | Specified collection metrics, optional `usage`，`iops` |
| cpu | object | The Configuration of cpu metrics collection. |
| cpu.metrics | []string | Specified collection metrics, optional `idle`，`system`，`user`，`iowait`，`load` |

## Node

The configuration of instance node.

Example:

```plain
node:
  major_ip_pattern: ".*"
  labels:
    env: dev
  tags:
    - linux
    - x86
    - es7
```

| Name | Type | Description |
| --- | --- | --- |
| major_ip_pattern | string | If there are multiple IPs on the host, use a pattern to control which IP is the primary one, which is used for reporting during registration. |
| labels | map | Custom lables |
| tags | []string | Custom tags |

## Misc

| Name                                    | Type   | Description                                           |
| --------------------------------------- | ------ | ----------------------------------------------------- |
| preference.pipeline_enabled_by_default   | map    | Whether pipelines are enabled by default. If set to `false`, each pipeline must be explicitly configured with `enabled` set to `true` |
| allow_multi_instance    | bool   | Whether is allowed to start multiple instances with the same program, default `false`                                                                                            |
| skip_instance_detect    | bool   | Whether is allowed to skip instance detection, default `false`                                                                                                          |
| max_num_of_instances    | int    | The maximum number of instances that the same program can run simultaneously, default `5`       |
