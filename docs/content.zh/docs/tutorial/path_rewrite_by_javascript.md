---
title: "使用 JavaScript 脚本来进行复杂的查询改写"
weight: 100
---

# 使用 JavaScript 脚本来进行复杂的查询改写

有这么一个需求：

> 网关里怎样对跨集群搜索进行支持的呢？我想实现: 输入的搜索请求是 `lp:9200/index1/_search`
> 这个索引在 3 个集群上，需要跨集群检索，也就是网关能否改成 `lp:9200/cluster01:index1,cluster02,index1,cluster03:index1/_search` 呢？
> 索引有一百多个，名称不一定是 app, 还可能多个索引一起的。

极限网关自带的过滤器 `content_regex_replace` 虽然可以实现字符正则替换，但是这个需求是带参数的变量替换，稍微复杂一点，没有办法直接用这个正则替换实现，有什么其他办法实现么？

## 使用脚本过滤器

当然有的，上面的这个需求，理论上我们只需要将其中的索引 `index1` 匹配之后，替换为 `cluster01:index1,cluster02,index1,cluster03:index1` 就行了。

答案就是使用自定义脚本来做，再复杂的业务逻辑都不是问题，都能通过自定义脚本来实现，一行脚本不行，那就两行。

使用极限网关提供的 [JavaScript](../references/filters/javascript/) 过滤器可以很灵活的实现这个功能，具体继续看。

## 定义脚本

首先创建一个脚本文件，放在网关数据目录的 `scripts` 子目录下面，如下：

```
➜  gateway ✗ tree data
data
└── gateway
    └── nodes
        └── c9bpg0ai4h931o4ngs3g
            ├── kvdb
            ├── queue
            ├── scripts
            │   └── index_path_rewrite.js
            └── stats
```

这个脚本的内容如下：

```
function process(context) {
    var originalPath = context.Get("_ctx.request.path");
    var matches = originalPath.match(/\/?(.*?)\/_search/)
    var indexNames = [];
    if(matches && matches.length > 1) {
        indexNames = matches[1].split(",")
    }
    var resultNames = []
    var clusterNames = ["cluster01", "cluster02"]
    if(indexNames.length > 0) {
        for(var i=0; i<indexNames.length; i++){
            if(indexNames[i].length > 0) {
                for(var j=0; j<clusterNames.length; j++){
                    resultNames.push(clusterNames[j]+":"+indexNames[i])
                }
            }
        }
    }

    if (resultNames.length>0){
        var newPath="/"+resultNames.join(",")+"/_search";
        context.Put("_ctx.request.path",newPath);
    }
}
```

和普通的 JavaScript 一样，定义一个特定的函数 `process` 来处理请求里面的上下文信息，`_ctx.request.path` 是网关内置上下文的一个变量，用来获取请求的路径，通过 `context.Get("_ctx.request.path")` 在脚本里面进行访问。

中间我们使用了 JavaScript 的正则匹配和字符处理，做了一些字符拼接，得到新的路径 `newPath` 变量，最后使用 `context.Put("_ctx.request.path",newPath)` 更新网关请求的路径信息，从而实现查询条件里面的参数替换。

有关网关内置上下文的变量列表，请访问 [Request Context](../references/context/)

## 定义网关

接下来，创建一个网关配置，并使用 `javascript` 过滤器调用该脚本，如下：

```
entry:
  - name: my_es_entry
    enabled: true
    router: my_router
    max_concurrency: 10000
    network:
      binding: 0.0.0.0:8000

flow:
  - name: default_flow
    filter:
      - dump:
          context:
            - _ctx.request.path
      - javascript:
          file: index_path_rewrite.js
      - dump:
          context:
          - _ctx.request.path
      - elasticsearch:
          elasticsearch: dev
router:
  - name: my_router
    default_flow: default_flow

elasticsearch:
- name: dev
  enabled: true
  schema: http
  hosts:
    - 192.168.3.188:9206
```

上面的例子中，使用了一个 `javascript` 过滤器，并且指定了加载的脚本文件为 `index_path_rewrite.js`，并使用了两个 `dump` 过滤器来输出脚本运行前后的路径信息，最后再使用一个 `elasticsearch` 过滤器来转发请求给 Elasticsearch 进行查询。

## 启动网关

我们启动网关测试一下，如下：

```
➜  gateway ✗ ./bin/gateway
   ___   _   _____  __  __    __  _
  / _ \ /_\ /__   \/__\/ / /\ \ \/_\ /\_/\
 / /_\///_\\  / /\/_\  \ \/  \/ //_\\\_ _/
/ /_\\/  _  \/ / //__   \  /\  /  _  \/ \
\____/\_/ \_/\/  \__/    \/  \/\_/ \_/\_/

[GATEWAY] A light-weight, powerful and high-performance elasticsearch gateway.
[GATEWAY] 1.0.0_SNAPSHOT, 2022-04-18 07:11:09, 2023-12-31 10:10:10, 8062c4bc6e57a3fefcce71c0628d2d4141e46953
[04-19 11:41:29] [INF] [app.go:174] initializing gateway.
[04-19 11:41:29] [INF] [app.go:175] using config: /Users/medcl/go/src/infini.sh/gateway/gateway.yml.
[04-19 11:41:29] [INF] [instance.go:72] workspace: /Users/medcl/go/src/infini.sh/gateway/data/gateway/nodes/c9bpg0ai4h931o4ngs3g
[04-19 11:41:29] [INF] [app.go:283] gateway is up and running now.
[04-19 11:41:30] [INF] [api.go:262] api listen at: http://0.0.0.0:2900
[04-19 11:41:30] [INF] [entry.go:312] entry [my_es_entry] listen at: http://0.0.0.0:8000
[04-19 11:41:30] [INF] [module.go:116] all modules are started
[04-19 11:41:30] [INF] [actions.go:349] elasticsearch [dev] is available
```

## 执行测试

运行下面的查询来验证查询结果，如下：

```
curl localhost:8000/abc,efg/_search
```

可以看到网关通过 `dump` 过滤器输出的调试信息：

```
---- DUMPING CONTEXT ----
_ctx.request.path  :  /abc,efg/_search
---- DUMPING CONTEXT ----
_ctx.request.path  :  /cluster01:abc,cluster02:abc,cluster01:efg,cluster02:efg/_search
```

查询条件按照我们的需求进行了改写，Nice！

## 重写 DSL 查询语句

好吧，我们刚刚只是修改了查询的索引而已，那么查询请求的 DSL 呢？行不行？

那自然是可以的嘛，瞧下面的例子:

```
function process(context) {
    var originalDSL = context.Get("_ctx.request.body");
    if (originalDSL.length >0){
        var jsonObj=JSON.parse(originalDSL);
        jsonObj.size=123;
        jsonObj.aggs= {
            "test1": {
                "terms": {
                    "field": "abc",
                        "size": 10
                }
            }
        }
        context.Put("_ctx.request.body",JSON.stringify(jsonObj));
    }
}
```

先是获取查询请求，然后转换成 JSON 对象，之后任意修改查询对象就行了，保存回去，搞掂。

测试一下:

```
 curl -XPOST   localhost:8000/abc,efg/_search -d'{"query":{}}'
```

输出:

```
---- DUMPING CONTEXT ----
_ctx.request.path  :  /abc,efg/_search
_ctx.request.body  :  {"query":{}}
[04-19 18:14:24] [INF] [reverseproxy.go:255] elasticsearch [dev] hosts: [] => [192.168.3.188:9206]
---- DUMPING CONTEXT ----
_ctx.request.path  :  /abc,efg/_search
_ctx.request.body  :  {"query":{},"size":123,"aggs":{"test1":{"terms":{"field":"abc","size":10}}}}
```

是不是感觉解锁了新的世界？

## 结论

通过使用 Javascript 脚本过滤器，我们可以非常灵活的进行复杂逻辑的操作来满足我们的业务需求。
