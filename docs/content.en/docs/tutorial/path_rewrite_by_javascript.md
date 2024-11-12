---
title: "Use JavaScript for complex query rewriting"
weight: 100
---

# Use JavaScript for complex query rewriting

Here is a use case：

> How does the gateway support cross-cluster search? I want to achieve: the input search request is `lp:9200/index1/_search`
> these indices are on three clusters, so need search across these clusters, how to use the gateways to switch to `lp:9200/cluster01:index1,cluster02,index1,cluster03:index1/_search`?
> we don't want to change the application side, there are more than 100 indices, the index name not strictly named as `index1`, may be multiple indices together。

Though INFINI Gateway provide a filter `content_regex_replace` can implement regular expression replacement,
but in this case the variable need to replace with multi parameters. It is more complex, there is no direct way to implement by regexp match and replace, so how do we do that?

## Javascript filter

The answer is yes, we do have a way, in the above case, in theory we only need to match the index name `index1` and replace 3 times by adding prefix `cluster01:`, `cluster02:` and `cluster03:`,

By using INFINI Gateway's [JavaScript](../references/filters/javascript/) filter, we can implement this easily.

Actually no matter how complex the business logic is, it can be implemented through the scripts, not one line of script, then two lines.

## Define the scripts

Let's create a script file under the `scripts` subdirectory of the gateway data directory, as follows:

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

The content of this script is as follows:

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

Like normal JavaScript, define a specific function `process` to handle context information inside the request,
`_ctx.request.path` is a variable of the gateway's built-in context to get the path of the request, and then use function `context.Get("_ctx.request.path")` to access this field inside the script.

In the script we used general regular expression for matching and characters process, did some character stitching, got a new path variable `newPath` , and finally used `context.Put("_ctx.request.path",newPath)` to update the request path back to context.

For more information about fields of request context please visit: [Request Context](../references/context/)

## Gateway Configuration

Next, create a gateway configuration and reference the script using a `javascript` filter as follows

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

In the above example, a `javascript` filter with file specified as `index_path_rewrite.js`, and two `dump` filters are used for debugging, also used one `elasticsearch` filter to forward requests to ElasticSearch for queries.

## Start Gateway

Let's start the gateway to have a test:

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

## Testing

Run the following query to verify the query results, as shown below:

```
curl localhost:8000/abc,efg/_search
```

You can see debugging information output by the gateway through the `dump` filter

```
---- DUMPING CONTEXT ----
_ctx.request.path  :  /abc,efg/_search
---- DUMPING CONTEXT ----
_ctx.request.path  :  /cluster01:abc,cluster02:abc,cluster01:efg,cluster02:efg/_search
```

The query criteria have been rewritten according to our requirements,Nice!

## Rewrite the DSL

All right, we did change the request url, is that also possible to change the request body, like the search QueryDSL?

Let's do this:

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

Testing:

```
 curl -XPOST   localhost:8000/abc,efg/_search -d'{"query":{}}'
```

Output:

```
---- DUMPING CONTEXT ----
_ctx.request.path  :  /abc,efg/_search
_ctx.request.body  :  {"query":{}}
[04-19 18:14:24] [INF] [reverseproxy.go:255] elasticsearch [dev] hosts: [] => [192.168.3.188:9206]
---- DUMPING CONTEXT ----
_ctx.request.path  :  /abc,efg/_search
_ctx.request.body  :  {"query":{},"size":123,"aggs":{"test1":{"terms":{"field":"abc","size":10}}}}
```

Look, we just unlock the new world, agree?

## Conclusion

By using the Javascript filter in INFINI Gateway, it can be very flexible and easily to perform the complex logical operations and rewrite the Elasticsearch QueryDSL to meet your business needs.
