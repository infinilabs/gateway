---
title: "其它配置"
weight: 250
---

# 其它配置

## 高级用法

### 配置模板

示例：

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

| 名称                        | 类型   | 说明                                                |
| --------------------------- | ------ | --------------------------------------------------- |
| configs.template            | array  | 配置模板，可以指定多个模板和对应的参数              |
| configs.template[].name     | string | 配置的名称                                          |
| configs.template[].path     | string | 模板配置路径                                        |
| configs.template[].variable | map    | 模板的参数设置，变量在模板里面的用法：`$[[变量名]]` |

### 使用环境变量

极限网关支持在配置里面使用环境变量来进行灵活的参数控制。

首先在配置里面定义环境变量的默认值，如下：

```
env:
  PROD_ES_ENDPOINT: http://localhost:9200
  PROD_ES_USER: elastic
  PROD_ES_PASS: password
```

然后就可以在配置里面通过如下语法来使用环境变量了：

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

注意，外部环境变量的优先级会大于配置内部的环境变量设置，比如希望在启动程序的时候覆盖环境变量，操作如下：

```
PROD_ES_ENDPOINT=http://1.1.1.1:9200 LOGGING_ES_ENDPOINT=http://2.2.2.2:9201  ./bin/gateway
```

## Path

配置、数据、日志相关路径配置。

示例：

```yaml
path.data: data
path.logs: log
path.configs: "config"
```

| 名称                    | 类型   | 说明                                                                                                                                            |
| ----------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| path.data               | string | 数据目录，默认为 `data`                                                                                                                         |
| path.logs               | string | 日志目录，默认为 `log`                                                                                                                          |
| path.configs            | string | 配置目录，默认为 `config`                                                                                                                       |

## Log

日志相关配置。

示例：

```yaml
log:
  level: info
  debug: false
```

| 名称                    | 类型   | 说明                                                                                                                                            |
| ----------------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| log.level               | string | 日志级别，默认为 `info`                                                                                                                         |
| log.debug               | bool   | 是否开启调试模式，当开启的时候，一旦出现异常程序直接退出，打印完整堆栈，仅用于调试定位故障点，默认为 `false`，生产环境不要开启，可能丢数据      |
| log.format              | bool   | 日志格式，默认为 `[%Date(01-02) %Time] [%LEV] [%File:%Line] %Msg%n`，[Format References](https://github.com/cihub/seelog/wiki/Format-reference) |
| log.disable_file_output | bool   | 是否关闭本地文件的日志输出，默认为 `false`，容器环境不希望本地日志输出的可以开启本参数                                                          |

## Configs

配置管理相关配置。

示例：

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

| 名称 | 类型 | 说明 |
| --- | --- | --- |
| configs.auto_reload | bool | 是否支持 `path.configs` 里面配置的动态加载 |
| configs.managed | bool | 是否支持由配置中心进行配置管理 |
| configs.servers | []string | 配置中心地址 |
| configs.interval | string | 配置同步间隔 |
| configs.soft_delete | bool | 配置文件删除为软删除，默认 `true` |
| configs.panic_on_config_error | bool | 配置加载如果有错误就直接崩溃，默认 `true` |
| configs.max_backup_files | int | 配置文件最大备份数，默认 `10` |
| configs.valid_config_extensions | []string | 有效的配置文件后缀，默认 `.tpl`, `.json`, `.yml`, `.yaml` |
| configs.tls | object | TLS 配置（请参考通用 [TLS](#tls-配置) 配置） |
| configs.always_register_after_restart | bool | 实例重启后是否进行注册，实例运行在 K8S 环境下，需开启此参数。 |
| configs.allow_generated_metrics_tasks | bool | 允许自动生成采集指标任务 |
| configs.ignored_path | []string | 需要忽略的配置文件路径 |                                                                                                    |

## 本地磁盘队列

示例：

```
disk_queue:
  upload_to_s3: true
  s3:
    server: my_blob_store
    location: cn-beijing-001
    bucket: infini-store
  max_bytes_per_file: 102400
```

| 名称                                        | 类型   | 说明                                                                               |
| ------------------------------------------- | ------ | ---------------------------------------------------------------------------------- |
| disk_queue.min_msg_size                     | int    | 发送到队列单条消息的最小字节限制，默认 `1`                                         |
| disk_queue.max_msg_size                     | int    | 发送到队列单条消息的最大字节限制，默认 `104857600`，即 100MB                       |
| disk_queue.sync_every_records               | int    | 每隔多少条记录进行一次 sync 磁盘同步操作，默认 `1000`                              |
| disk_queue.sync_timeout_in_ms               | int    | 每隔多长时间进行一次 sync 磁盘同步操作，默认 `1000` 毫秒                           |
| disk_queue.max_bytes_per_file               | int    | 本地磁盘队列单个文件的最大值，超过此大小自动滚动新文件，默认 `104857600`，即 100MB |
| disk_queue.max_used_bytes                   | int    | 本地磁盘队列可允许的最大存储使用空间大小                                           |
| disk_queue.warning_free_bytes               | int    | 磁盘达到告警阈值的空闲存储空间大小，默认 `10737418240` 即 10GB                     |
| disk_queue.reserved_free_bytes              | int    | 磁盘空闲存储空间大小的保护值，达到会变成只读，不允许写，默认 `5368709120` 即 5GB   |
| disk_queue.auto_skip_corrupted_file                     | bool   | 是否自动跳过损坏的磁盘文件，默认 `true`                                          |
| disk_queue.upload_to_s3                     | bool   | 是否将磁盘队列文件上传到 S3，默认 `false`                                          |
| disk_queue.s3.async                         | bool   | 是否异步上传到 S3 服务器                                                           |
| disk_queue.s3.server                        | string | S3 服务器 ID                                                                       |
| disk_queue.s3.location                      | string | S3 服务器位置                                                                      |
| disk_queue.s3.bucket                        | string | S3 服务器 Bucket                                                                   |
| disk_queue.retention.max_num_of_local_files | int    | 上传 s3 完的文件，按照最新的文件排序，保留在本地磁盘上的最大文件数，默认 `3`      |
| disk_queue.compress.segment.enabled         | bool   | 是否开启文件级别的压缩，默认 `false`                                               |

## S3 

示例：

```
s3:
  my_blob_store:
    endpoint: "192.168.3.188:9000"
    access_key: "admin"
    access_secret: "gogoaminio"
```

| 名称                         | 类型   | 说明                    |
| ---------------------------- | ------ | ----------------------- |
| s3.[id].endpoint             | string | S3 服务器地址           |
| s3.[id].access_key           | string | S3 服务器 Key           |
| s3.[id].access_secret        | string | S3 服务器秘钥           |
| s3.[id].token                | string | S3 服务器 Token 信息    |
| s3.[id].ssl                  | bool   | S3 服务器是否开启了 TLS |
| s3.[id].skip_insecure_verify | bool   | 是否忽略 TLS 证书校验   |

## Kafka

极限网关支持在使用分布式 Kafka 作为后端队列，相关参数如下。

| 名称                       | 类型   | 说明                                                         |
| -------------------------- | ------ | ------------------------------------------------------------ |
|   kafka.enabled      | bool    | Kafka 模块是否开启                                                |
|   kafka.default      | bool    | Kafka 模块是否作为默认 Queue 的实现                                                |
|   kafka.num_of_partition      | int    | 默认的分区数量，默认 `1`                                               |
|   kafka.num_of_replica      | int    | 默认的分区副本数量，默认 `1`                                               |
|   kafka.producer_batch_max_bytes      | int    | 最大提交请求大小，默认 `50 * 1024 * 1024`                                               |
|   kafka.max_buffered_records      | int    | 最大缓存请求记录数，默认 `10000`                                               |
|   kafka.manual_flushing      | bool    | 是否手动 flushing，默认 `false`                                               |
|   kafka.brokers      | []string    | 服务器地址信息                                               |
|   kafka.username      | string    | 用户信息                                            |
|   kafka.password      | string    | 密码信息                                            |

## Badger

Badger 是一个轻量级的基于磁盘的 KeyValue 存储引擎，极限网关使用 Badger 来实现 KV 模块的存储。

| 名称                       | 类型   | 说明                                                         |
| -------------------------- | ------ | ------------------------------------------------------------ |
|   badger.enabled      | bool    |   是否启用 Badger实现的 KV 模块，默认为 `true`  |
|   badger.single_bucket_mode      | bool    |   Badger 模块使用单桶模式，默认为 `true`  |
|   badger.sync_writes      | bool    |   Badger 模块使用同步写，默认为 `false`  |
|   badger.mem_table_size      | int64    |   Badger 模块的内存表大小，默认为 `10 * 1024 * 1024`，即 `10485760` |
|   badger.value_log_file_size      | int64    |   Badger 模块的日志文件大小，默认为 `1<<30 - 1`，即 1g |
|   badger.value_log_max_entries      | int64    |   Badger 模块的日志消息个数，默认为 `1000000`，即 1million |
|   badger.value_threshold      | int64    |   Badger 模块的值大小阈值，默认为 `1048576`，即 1m |
|   badger.num_mem_tables      | int64    |   Badger 模块的内存表个数，默认为 `1`|
|   badger.num_level0_tables      | int64    |   Badger 模块的 Level0 内存表个数，默认为 `1`|

## 资源限制

| 名称                       | 类型   | 说明                                                         |
| -------------------------- | ------ | ------------------------------------------------------------ |
|   resource_limit.cpu.max_num_of_cpus      | int    | 允许使用的最大 CPU 核数，仅用于 Linux 操作系统，且 `taskset` 命令可用   |
|   resource_limit.cpu.affinity_list      | string    | 允许使用的 CPU 绑定设置，eg: `0,2,5` 或 `0-8`，仅用于 Linux 操作系统，且 `taskset` 命令可用   |
| resource_limit.memory.max_in_bytes   | string | 允许使用的内存的最大大小，软性限制           |



## 网络配置

公共的网络配置说明。

| 名称                       | 类型   | 说明                                                         |
| -------------------------- | ------ | ------------------------------------------------------------------- |
| *.network.host               | string | 服务监听的网络地址，例如，`192.168.3.10`              |
| *.network.port               | int    | 服务监听的端口地址，例如，`8000`                         |
| *.network.binding            | string | 服务监听的网络绑定地址，例如，`0.0.0.0:8000`      |
| *.network.publish            | string | 服务监听的外部访问地址，例如，`192.168.3.10:8000` |
| *.network.reuse_port         | bool   | 是否在多进程端口共享中重用网络端口                     |
| *.network.skip_occupied_port | bool   | 是否自动跳过已占用的端口                                         |

## TLS 配置

公共的 TLS 配置说明。

| 名称                       | 类型   | 说明                                                         |
| -------------------------- | ------ | ------------------------------------------------------------------- |
| *.tls.enabled                | bool   | 是否启用 TLS 安全传输，不指定证书可自动生成                                |
| *.tls.ca_file              | string | TLS 安全证书的公共 CA 证书路径                                       |
| *.tls.cert_file              | string | TLS 安全证书的公共密钥路径                                       |
| *.tls.key_file               | string | TLS 安全证书的私钥路径                                          |
| *.tls.skip_insecure_verify   | bool   | 是否忽略 TLS 证书验证                                              |
| *.tls.default_domain   | string   | 用于自动生成证书的默认域名                                  |
| *.tls.skip_domain_verify   | bool   | 是否跳过域名验证                                              |
| *.tls.client_session_cache_size   | int   | 设置 TLS 会话恢复的最大客户端会话状态缓存大小   |

## API

| 名称                       | 类型   | 说明                                                         |
| -------------------------- | ------ | ------------------------------------------------------------------- |
| api.enabled | bool    | 是否启用 API 模块， 默认为 `true`           |
| api.network | object    | 网络配置，请参考通用网络配置部分         |
| api.tls | object    | TLS 配置，请参考通用 TLS 配置部分        |
| api.security | object    | API 模块的安全配置     |
| api.security.enabled | bool    | 是否启用安全性     |
| api.security.username | string    | 安全性的用户名    |
| api.security.password | string    | 安全性的密码   |
| api.cors.allowed_origins | []string    | 跨域请求可以执行的源列表   |
| api.websocket | object    | API 模块的 WebSocket 配置 |
| api.websocket.enabled | object    | 是否启用 WebSocket     |
| api.websocket.permitted_hosts | []string    | 允许访问 WebSocket 服务的主机列表 |
| api.websocket.skip_host_verify | bool    | 是否跳过验证 WebSocket 的主机 |

## Metrics

配置系统指标采集。

示例：

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

| 名称 | 类型 | 说明 |
| --- | --- | --- |
| enabled | bool | 是否开启系统指标采集，默认 `true` |
| queue | string | 指标采集队列 |
| network | object | 采集网络指标配置 |
| network.enabled | bool | 是否采集网络指标，默认 `true` |
| network.summary | bool | 是否采集 summary 指标 |
| network.sockets | bool | 是否采集相关 socket 指标 |
| network.throughput | bool | 是否采集 throughput 指标 |
| network.details | bool | 是否将网络 IO 指标进行累计 |
| network.interfaces | []string | 指定需要采集的网络接口，默认采集所有接口 |
| memory | object | 采集内存指标配置 |
| memory.enabled | bool | 是否开启内存指标的采集，默认 `true` |
| memory.metrics | []string | 指定采集相关指标，可选 `swap`，`memory` |
| disk | object | 采集磁盘指标配置 |
| disk.metrics | []string | 指定采集相关指标，可选 `usage`，`iops` |
| cpu | object | 采集 CPU 指标配置 |
| cpu.metrics | []string | 指定采集相关指标，可选 `idle`，`system`，`user`，`iowait`，`load` |

## Node

节点相关配置。

示例：

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

| 名称 | 类型 | 说明 |
| --- | --- | --- |
| major_ip_pattern | string | 如果主机上有多个 IP，用 pattern 来控制以哪个 IP 为主，当注册时用于上报。 |
| labels | map | 自定义标签 |
| tags | []string | 自定义标签 |

## 其它配置

| 名称                       | 类型   | 说明                                                         |
| -------------------------- | ------ | ------------------------------------------------------------ |
|   preference.pipeline_enabled_by_default      | map    | Pipeline 是否默认启动，如果改成 `false`，则需要每个 Pipeline 配置显式设置 `enabled` 为 `true`                                                 |
| allow_multi_instance    | bool   | 是否允许同一程序启动多个实例，默认为 `false`                                                                                            |
| skip_instance_detect    | bool   | 是否跳过实例检测，默认为 `false`                                                                                                          |
| max_num_of_instances    | int    | 同一程序可同时运行的实例的最大个数，默认为 `5`                                                                                                                  |