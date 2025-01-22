---
weight: 50
title: "性能测试"
---

# 性能测试

推荐使用 Elasticsearch 专属压测工具 `Loadgen` 来对网关进行性能压测。

Loadgen 的特点：

- 性能强劲
- 轻量级无依赖
- 支持模板化参数随机
- 支持高并发
- 支持压测端均衡流量控制
- 支持服务端返回值校验

> 下载地址：<https://release.infinilabs.com/loadgen/>

## Loadgen

Loadgen 使用非常简单，下载解压之后会得到三个文件，一个可执行程序、一个配置文件 `loadgen.yml` 以及用于运行测试的 `loadgen.dsl`，配置文件样例如下：

```yaml
env:
  ES_USERNAME: elastic
  ES_PASSWORD: elastic
  ES_ENDPOINT: http://localhost:8000
```

测试文件样例如下：

```text
# runner: {
#   // total_rounds: 1
#   no_warm: false,
#   // Whether to log all requests
#   log_requests: false,
#   // Whether to log all requests with the specified response status
#   log_status_codes: [0, 500],
#   assert_invalid: false,
#   assert_error: false,
# },
# variables: [
#   {
#     name: "ip",
#     type: "file",
#     path: "dict/ip.txt",
#     // Replace special characters in the value
#     replace: {
#       '"': '\\"',
#       '\\': '\\\\',
#     },
#   },
#   {
#     name: "id",
#     type: "sequence",
#   },
#   {
#     name: "id64",
#     type: "sequence64",
#   },
#   {
#     name: "uuid",
#     type: "uuid",
#   },
#   {
#     name: "now_local",
#     type: "now_local",
#   },
#   {
#     name: "now_utc",
#     type: "now_utc",
#   },
#   {
#     name: "now_utc_lite",
#     type: "now_utc_lite",
#   },
#   {
#     name: "now_unix",
#     type: "now_unix",
#   },
#   {
#     name: "now_with_format",
#     type: "now_with_format",
#     // https://programming.guide/go/format-parse-string-time-date-example.html
#     format: "2006-01-02T15:04:05-0700",
#   },
#   {
#     name: "suffix",
#     type: "range",
#     from: 10,
#     to: 1000,
#   },
#   {
#     name: "bool",
#     type: "range",
#     from: 0,
#     to: 1,
#   },
#   {
#     name: "list",
#     type: "list",
#     data: ["medcl", "abc", "efg", "xyz"],
#   },
#   {
#     name: "id_list",
#     type: "random_array",
#     variable_type: "number", // number/string
#     variable_key: "suffix", // variable key to get array items
#     square_bracket: false,
#     size: 10, // how many items for array
#   },
#   {
#     name: "str_list",
#     type: "random_array",
#     variable_type: "number", // number/string
#     variable_key: "suffix", // variable key to get array items
#     square_bracket: true,
#     size: 10, // how many items for array
#     replace: {
#       // Use ' instead of " for string quotes
#       '"': "'",
#       // Use {} instead of [] as array brackets
#       "[": "{",
#       "]": "}",
#     },
#   },
# ],

POST $[[env.ES_ENDPOINT]]/medcl/_search
{ "track_total_hits": true, "size": 0, "query": { "terms": { "patent_id": [ $[[id_list]] ] } } }
# request: {
#   runtime_variables: {batch_no: "uuid"},
#   runtime_body_line_variables: {routing_no: "uuid"},
#   basic_auth: {
#     username: "$[[env.ES_USERNAME]]",
#     password: "$[[env.ES_PASSWORD]]",
#   },
# },
```

### 运行模式设置

默认配置下，Loadgen 会以性能测试模式运行，在指定时间（`-d`）内重复执行 `requests` 里的所有请求。如果只需要检查一次测试结果，可以通过 `runner.total_rounds` 来设置 `requests` 的执行次数。

### HTTP 响应头处理

默认配置下，Loadgen 会自动格式化 HTTP 的响应头（`user-agent: xxx` -> `User-Agent: xxx`），如果需要精确判断服务器返回的响应头，可以通过 `runner.disable_header_names_normalizing` 来禁用这个行为。

## 变量的使用

上面的配置中，`variables` 用来定义变量参数，根据 `name` 来设置变量标识，在构造请求的使用 `$[[变量名]]` 即可访问该变量的值，变量目前支持的类型有：

| 类型              | 说明                                                                                 | 变量参数                                                                                                                                                                   |
| ----------------- | ------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `file`            | 文件型外部变量参数                                                                   | `path`: 数据文件路径<br>`data`: 数据列表，会被附加到`path`文件内容后读取                                                                                                   |
| `list`            | 自定义枚举变量参数                                                                   | `data`: 字符数组类型的枚举数据列表                                                                                                                                         |
| `sequence`        | 32 位自增数字类型的变量                                                              | `from`: 初始值<br>`to`: 最大值                                                                                                                                             |
| `sequence64`      | 64 位自增数字类型的变量                                                              | `from`: 初始值<br>`to`: 最大值                                                                                                                                             |
| `range`           | 数字范围类型的变量，支持参数 `from` 和 `to` 来限制范围                               | `from`: 初始值<br>`to`: 最大值                                                                                                                                             |
| `random_array`    | 生成一个随机数组，数据元素来自`variable_key`指定的变量                               | `variable_key`: 数据源变量<br>`size`: 输出数组的长度<br>`square_bracket`: `true/false`，输出值是否需要`[`和`]`<br>`string_bracket`: 字符串，输出元素前后会附加指定的字符串 |
| `uuid`            | UUID 字符类型的变量                                                                  |                                                                                                                                                                            |
| `now_local`       | 当前时间、本地时区                                                                   |                                                                                                                                                                            |
| `now_utc`         | 当前时间、UTC 时区。输出格式:`2006-01-02 15:04:05.999999999 -0700 MST`               |                                                                                                                                                                            |
| `now_utc_lite`    | 当前时间、UTC 时区。输出格式:`2006-01-02T15:04:05.000`                               |                                                                                                                                                                            |
| `now_unix`        | 当前时间、Unix 时间戳                                                                |                                                                                                                                                                            |
| `now_with_format` | 当前时间，支持自定义 `format` 参数来格式化时间字符串，如：`2006-01-02T15:04:05-0700` | `format`: 输出的时间格式 ([示例](https://www.geeksforgeeks.org/time-formatting-in-golang/))                                                                                |

### 变量使用示例

`file` 类型变量参数加载自外部文本文件，每行一个变量参数，访问该变量时每次随机取其中一个，变量里面的定义格式举例如下：

```text
# test/user.txt
medcl
elastic
```

附生成固定长度的随机字符串，如 1024 个字符每行：

```bash
LC_CTYPE=C tr -dc A-Za-z0-9_\!\@\#\$\%\^\&\*\(\)-+= < /dev/random | head -c 1024 >> 1k.txt
```

### 环境变量

Loadgen 支持自动读取环境变量，环境变量可以在运行 Loadgen 时通过命令行传入，也可以在 `loadgen.dsl` 里指定默认的环境变量值，Loadgen 运行时会使用命令行传入的环境变量覆盖 `loadgen.dsl` 里的默认值。

配置的环境变量可以通过 `$[[env.环境变量]]` 来使用：

```text
#// 配置环境变量默认值
# env: {
#   ES_USERNAME: "elastic",
#   ES_PASSWORD: "elastic",
#   ES_ENDPOINT: "http://localhost:8000",
# },

#// 使用运行时变量
GET $[[env.ES_ENDPOINT]]/medcl/_search
{"query": {"match": {"name": "$[[user]]"}}}
# request: {
#   // 使用运行时变量
#   basic_auth: {
#     username: "$[[env.ES_USERNAME]]",
#     password: "$[[env.ES_PASSWORD]]",
#   },
# },
```

## 请求的定义

配置节点 `requests` 用来设置 Loadgen 将依次执行的请求，支持固定参数的请求，也可支持模板变量参数化构造请求，以下是一个普通的查询请求：

```text
GET http://localhost:8000/medcl/_search?q=name:$[[user]]
# request: {
#   username: elastic,
#   password: pass,
# },
```

上面的查询对 `medcl` 索引进行了查询，并对 `name` 字段执行一个查询，每次请求的值来自随机变量 `user`。

### 模拟批量写入

使用 Loadgen 来模拟 bulk 批量写入也非常简单，在请求体里面配置一条索引操作，然后使用 `body_repeat_times` 参数来随机参数化复制若干条请求即可完成一批请求的准备，如下：

```text
POST http://localhost:8000/_bulk
{"index": {"_index": "medcl-y4", "_type": "doc", "_id": "$[[uuid]]"}}
{"id": "$[[id]]", "field1": "$[[user]]", "ip": "$[[ip]]", "now_local": "$[[now_local]]", "now_unix": "$[[now_unix]]"}
# request: {
#   basic_auth: {
#     username: "test",
#     password: "testtest",
#  },
#  body_repeat_times: 1000,
# },
```

### 返回值判断

每个 `requests` 配置可以通过 `assert` 来设置是否需要检查返回值。`assert` 功能支持 INFINI Gateway 的大部分[条件判断功能](../references/flow/#条件判断)。

> 请阅读[《借助 DSL 来简化 Loadgen 配置》](https://infinilabs.cn/blog/2023/simplify-loadgen-config-with-dsl/)来了解更多细节。

```text
GET http://localhost:8000/medcl/_search?q=name:$[[user]]
# request: {
#   basic_auth: {
#     username: "test",
#     password: "testtest",
#  },
# },
# assert: {
#   _ctx.response.status: 201,
# },
```

请求返回值可以通过 `_ctx` 获取，`_ctx` 目前包含以下信息：

| 参数                      | 说明                                                                                    |
| ------------------------- | --------------------------------------------------------------------------------------- |
| `_ctx.response.status`    | HTTP 返回状态码                                                                         |
| `_ctx.response.header`    | HTTP 返回响应头                                                                         |
| `_ctx.response.body`      | HTTP 返回响应体                                                                         |
| `_ctx.response.body_json` | 如果 HTTP 返回响应体是一个有效的 JSON 字符串，可以通过 `body_json` 来访问 JSON 内容字段 |
| `_ctx.elapsed`            | 当前请求发送到返回消耗的时间（毫秒）                                                    |

如果请求失败（请求地址无法访问等），Loadgen 无法获取 HTTP 请求返回值，Loadgen 会在输出日志里记录 `Number of Errors`。如果配置了 `runner.assert_error` 且存在请求失败的请求，Loadgen 会返回 `exit(2)` 错误码。

如果返回值不符合判断条件，Loadgen 会停止执行当前轮次后续请求，并在输出日志里记录 `Number of Invalid`。如果配置了 `runner.assert_invalid` 且存在判断失败的请求，Loadgen 会返回 `exit(1)` 错误码。

### 动态变量注册

每个 `requests` 配置可以通过 `register` 来动态添加运行时参数，一个常见的使用场景是根据前序请求的返回值来动态设置后序请求的参数。

这个示例调用 `$[[env.ES_ENDPOINT]]/test` 接口获取索引的 UUID，并注册到 `index_id` 变量。后续的请求定义可以通过 `$[[index_id]]` 来获取这个值。

```text
GET $[[env.ES_ENDPOINT]]/test
# register: [
#   {index_id: "_ctx.response.body_json.test.settings.index.uuid"},
# ],
# assert: (200, {}),
```

## 执行压测

执行 Loadgen 程序即可执行压测，如下:

```text
$ loadgen -d 30 -c 100 -compress -run loadgen.dsl
   __   ___  _      ___  ___   __    __
  / /  /___\/_\    /   \/ _ \ /__\/\ \ \
 / /  //  ///_\\  / /\ / /_\//_\ /  \/ /
/ /__/ \_//  _  \/ /_// /_\\//__/ /\  /
\____|___/\_/ \_/___,'\____/\__/\_\ \/

[LOADGEN] A http load generator and testing suit.
[LOADGEN] 1.0.0_SNAPSHOT, 83f2cb9, Sun Jul 4 13:52:42 2021 +0800, medcl, support single item in dict files
[07-19 16:15:00] [INF] [instance.go:24] workspace: data/loadgen/nodes/0
[07-19 16:15:00] [INF] [loader.go:312] warmup started
[07-19 16:15:00] [INF] [app.go:306] loadgen now started.
[07-19 16:15:00] [INF] [loader.go:316] [GET] http://localhost:8000/medcl/_search
[07-19 16:15:00] [INF] [loader.go:317] status: 200,<nil>,{"took":1,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":0,"relation":"eq"},"max_score":null,"hits":[]}}
[07-19 16:15:00] [INF] [loader.go:316] [GET] http://localhost:8000/medcl/_search?q=name:medcl
[07-19 16:15:00] [INF] [loader.go:317] status: 200,<nil>,{"took":1,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":0,"relation":"eq"},"max_score":null,"hits":[]}}
[07-19 16:15:01] [INF] [loader.go:316] [POST] http://localhost:8000/_bulk
[07-19 16:15:01] [INF] [loader.go:317] status: 200,<nil>,{"took":120,"errors":false,"items":[{"index":{"_index":"medcl-y4","_type":"doc","_id":"c3qj9123r0okahraiej0","_version":1,"result":"created","_shards":{"total":2,"successful":1,"failed":0},"_seq_no":5735852,"_primary_term":3,"status":201}}]}
[07-19 16:15:01] [INF] [loader.go:325] warmup finished

5253 requests in 32.756483336s, 524.61KB sent, 2.49MB received

[Loadgen Client Metrics]
Requests/sec:		175.10
Request Traffic/sec:	17.49KB
Total Transfer/sec:	102.34KB
Avg Req Time:		5.711022ms
Fastest Request:	440.448µs
Slowest Request:	3.624302658s
Number of Errors:	0
Number of Invalid:	0
Status 200:		5253

[Estimated Server Metrics]
Requests/sec:		160.37
Transfer/sec:		93.73KB
Avg Req Time:		623.576686ms
```

Loadgen 在正式压测之前会将所有的请求执行一次来进行预热，如果出现错误会提示是否继续，预热的请求结果也会输出到终端，执行完成之后会输出执行的摘要信息。可以通过设置 `runner.no_warm` 来跳过这个检查阶段。

> 因为 Loadgen 最后的结果是所有请求全部执行完成之后的累计统计，可能存在不准的问题，建议通过打开 Kibana 的监控仪表板来实时查看 Elasticsearch 的各项运行指标。

### 命令行参数

Loadgen 会循环执行配置文件里面定义的请求，默认 Loadgen 只会运行 `5s` 就自动退出了，如果希望延长运行时间或者加大并发可以通过启动的时候设置参数来控制，通过查看帮助命令如下：

```text
$ loadgen -help
Usage of loadgen:
  -c int
    	Number of concurrent threads (default 1)
  -compress
    	Compress requests with gzip
  -config string
    	the location of config file (default "loadgen.yml")
  -cpu int
    	the number of CPUs to use (default -1)
  -d int
    	Duration of tests in seconds (default 5)
  -debug
    	run in debug mode, loadgen will quit on panic immediately with full stack trace
  -dial-timeout int
    	Connection dial timeout in seconds, default 3s (default 3)
  -gateway-log string
    	Log level of Gateway (default "debug")
  -l int
    	Limit total requests (default -1)
  -log string
    	the log level, options: trace,debug,info,warn,error,off
  -mem int
    	the max size of Memory to use, soft limit in megabyte (default -1)
  -plugin value
    	load additional plugins
  -r int
    	Max requests per second (fixed QPS) (default -1)
  -read-timeout int
    	Connection read timeout in seconds, default 0s (use -timeout)
  -run string
    	DSL config to run tests (default "loadgen.dsl")
  -service string
    	service management, options: install,uninstall,start,stop
  -timeout int
    	Request timeout in seconds, default 60s (default 60)
  -v	version
  -write-timeout int
    	Connection write timeout in seconds, default 0s (use -timeout)
```

### 限制客户端压力

使用 Loadgen 并设置命令行参数 `-r` 可以限制客户端发送的每秒请求数，从而评估固定压力下 Elasticsearch 的响应时间和负载情况，如下：

```bash
loadgen -d 30 -c 100 -r 100
```

> 注意，在大量并发下，此客户端吞吐限制可能不完全准确。

### 限制请求的总条数

通过设置参数 `-l` 可以控制客户端发送的请求总数，从而制造固定的文档，修改配置如下：

```text
#// loadgen-gw.dsl
POST http://localhost:8000/medcl-test/doc2/_bulk
{"index": {"_index": "medcl-test", "_id": "$[[uuid]]"}}
{"id": "$[[id]]", "field1": "$[[user]]", "ip": "$[[ip]]"}
# request: {
#   basic_auth: {
#     username: "test",
#     password: "testtest",
#   },
#   body_repeat_times: 1,
# },
```

每次请求只有一个文档，然后执行 Loadgen

```bash
loadgen -run loadgen-gw.dsl -d 600 -c 100 -l 50000
```

执行完成之后，Elasticsearch 的索引 `medcl-test` 将增加 `50000` 条记录。

### 使用自增 ID 来确保文档的顺序性

如果希望生成的文档编号自增有规律，方便进行对比，可以使用 `sequence` 类型的自增 ID 来作为主键，内容也不要用随机数，如下：

```text
POST http://localhost:8000/medcl-test/doc2/_bulk
{"index": {"_index": "medcl-test", "_id": "$[[id]]"}}
{"id": "$[[id]]"}
# request: {
#   basic_auth: {
#     username: "test",
#     password: "testtest",
#   },
#   body_repeat_times: 1,
# },
```

### 上下文复用变量

在一个请求中，我们可能希望有相同的参数出现，比如 `routing` 参数用来控制分片的路由，同时我们又希望该参数也保存在文档的 JSON 里面，
可以使用 `runtime_variables` 来设置请求级别的变量，或者 `runtime_body_line_variables` 定义请求体级别的变量，如果请求体复制 N 份，每份的参数是不同的，举例如下：

```text
# variables: [
#   {name: "id", type: "sequence"},
#   {name: "uuid", type: "uuid"},
#   {name: "now_local", type: "now_local"},
#   {name: "now_utc", type: "now_utc"},
#   {name: "now_unix", type: "now_unix"},
#   {name: "suffix", type: "range", from: 10, to 15},
# ],

POST http://192.168.3.188:9206/_bulk
{"create": {"_index": "test-$[[suffix]]", "_type": "doc", "_id": "$[[uuid]]", "routing": "$[[routing_no]]"}}
{"id": "$[[uuid]]", "routing_no": "$[[routing_no]]", "batch_number": "$[[batch_no]]", "random_no": "$[[suffix]]", "ip": "$[[ip]]", "now_local": "$[[now_local]]", "now_unix": "$[[now_unix]]"}
# request: {
#   runtime_variables: {
#     batch_no: "id",
#   },
#   runtime_body_line_variables: {
#     routing_no: "uuid",
#   },
#   basic_auth: {
#     username: "ingest",
#     password: "password",
#   },
#   body_repeat_times: 10,
# },
```

我们定义了 `batch_no`　变量来代表一批文档里面的相同批次号，同时又定义了　`routing_no`　变量来代表每个文档级别的 routing 值。

### 自定义 Header

```text
GET http://localhost:8000/test/_search
# request: {
#   headers: [
#     {Agent: "Loadgen-1"},
#   ],
#   disable_header_names_normalizing: false,
# },
```

默认配置下，Loadgen 会自动格式化配置里的 HTTP 的请求头（`user-agent: xxx` -> `User-Agent: xxx`），如果需要精确设置 HTTP 请求头，可以通过设置 `disable_header_names_normalizing: true` 来禁用这个行为。

## 运行测试套件

Loadgen 支持批量运行测试用例，不需要重复编写测试用例，通过切换套件配置来快速测试不同的环境配置：

```yaml
# loadgen.yml
env:
  # Set up envrionments to run test suite
  LR_TEST_DIR: ./testing # The path to the test cases.
  # If you want to start gateway dynamically and automatically:
  LR_GATEWAY_CMD: ./bin/gateway # The path to the executable of INFINI Gateway
  LR_GATEWAY_HOST: 0.0.0.0:18000 # The binding host of the INFINI Gateway
  LR_GATEWAY_API_HOST: 0.0.0.0:19000 # The binding host of the INFINI Gateway API server
  # Set up other envrionments for the gateway and loadgen
  LR_ELASTICSEARCH_ENDPOINT: http://localhost:19201
  CUSTOM_ENV: myenv
tests:
  # The relative path of test cases under `LR_TEST_DIR`
  #
  # - gateway.yml: (Optional) the configuration to start the INFINI Gateway dynamically.
  # - loadgen.dsl: the configuration to run the loadgen tool.
  #
  # The environments set in `env` section will be passed to the INFINI Gateway and loadgen.
  - path: cases/gateway/echo/echo_with_context
```

### 环境变量配置

Loadgen 通过环境变量来动态配置 INFINI Gateway，环境变量在 `env` 里指定。以下环境变量是必选的：

| 变量名        | 说明             |
| ------------- | ---------------- |
| `LR_TEST_DIR` | 测试用例所在目录 |

如果你需要 `loadgen` 根据配置动态启动 INFINI Gateway，需要设置以下环境变量：

| 变量名                | 说明                                 |
| --------------------- | ------------------------------------ |
| `LR_GATEWAY_CMD`      | INFINI Gateway 可执行文件的路径      |
| `LR_GATEWAY_HOST`     | INFINI Gateway 绑定的主机名:端口     |
| `LR_GATEWAY_API_HOST` | INFINI Gateway API 绑定的主机名:端口 |

### 测试用例配置

测试用例在 `tests` 里配置，每个路径（`path`）指向一个测试用例的目录，每个测试用例需要配置一份 `gateway.yml`（可选）和 `loadgen.dsl`。配置文件可以使用 `env` 下配置的环境变量（`$[[env.ENV_KEY]]`）。

`gateway.yml` 参考配置：

```yaml
path.data: data
path.logs: log

entry:
  - name: my_es_entry
    enabled: true
    router: my_router
    max_concurrency: 200000
    network:
      binding: $[[env.LR_GATEWAY_HOST]]

flow:
  - name: hello_world
    filter:
      - echo:
          message: "hello world"
router:
  - name: my_router
    default_flow: hello_world
```

`loadgen.dsl` 参考配置：

```
# runner: {
#   total_rounds: 1,
#   no_warm: true,
#   log_requests: true,
#   assert_invalid: true,
#   assert_error: true,
# },

GET http://$[[env.LR_GATEWAY_HOST]]/
# assert: {
#   _ctx.response: {
#     status: 200,
#     body: "hello world",
#   },
# },
```

### 测试套件运行

配置好测试 `loadgen.yml` 后，可以通过以下命令运行 Loadgen：

```bash
loadgen -config loadgen.yml
```

Loadgen 会运行配置指定的所有测试用例，并输出测试结果：

```text
$ loadgen -config loadgen.yml
   __   ___  _      ___  ___   __    __
  / /  /___\/_\    /   \/ _ \ /__\/\ \ \
 / /  //  ///_\\  / /\ / /_\//_\ /  \/ /
/ /__/ \_//  _  \/ /_// /_\\//__/ /\  /
\____|___/\_/ \_/___,'\____/\__/\_\ \/

[LOADGEN] A http load generator and testing suit.
[LOADGEN] 1.0.0_SNAPSHOT, 83f2cb9, Sun Jul 4 13:52:42 2021 +0800, medcl, support single item in dict files
[02-21 10:50:05] [INF] [app.go:192] initializing loadgen
[02-21 10:50:05] [INF] [app.go:193] using config: /Users/kassian/Workspace/infini/src/infini.sh/testing/suites/dev.yml
[02-21 10:50:05] [INF] [instance.go:78] workspace: /Users/kassian/Workspace/infini/src/infini.sh/testing/data/loadgen/nodes/cfpihf15k34iqhpd4d00
[02-21 10:50:05] [INF] [app.go:399] loadgen is up and running now.
[2023-02-21 10:50:05][TEST][SUCCESS] [setup/loadgen/cases/dummy] duration: 105(ms)

1 requests in 68.373875ms, 0.00bytes sent, 0.00bytes received

[Loadgen Client Metrics]
Requests/sec:		0.20
Request Traffic/sec:	0.00bytes
Total Transfer/sec:	0.00bytes
Avg Req Time:		5s
Fastest Request:	68.373875ms
Slowest Request:	68.373875ms
Number of Errors:	0
Number of Invalid:	0
Status 200:		1

[Estimated Server Metrics]
Requests/sec:		14.63
Transfer/sec:		0.00bytes
Avg Req Time:		68.373875ms


[2023-02-21 10:50:06][TEST][FAILED] [setup/gateway/cases/echo/echo_with_context/] duration: 1274(ms)
#0 request, GET http://$[[env.LR_GATEWAY_HOST]]/any/, assertion failed, skiping subsequent requests
1 requests in 1.255678s, 0.00bytes sent, 0.00bytes received

[Loadgen Client Metrics]
Requests/sec:		0.20
Request Traffic/sec:	0.00bytes
Total Transfer/sec:	0.00bytes
Avg Req Time:		5s
Fastest Request:	1.255678s
Slowest Request:	1.255678s
Number of Errors:	1
Number of Invalid:	1
Status 0:		1

[Estimated Server Metrics]
Requests/sec:		0.80
Transfer/sec:		0.00bytes
Avg Req Time:		1.255678s

```
