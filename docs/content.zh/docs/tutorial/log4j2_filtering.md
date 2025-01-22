---
title: "Apache Log4j 漏洞处置"
weight: 10
---

# Apache Log4j 漏洞处置

【CVE 地址】

[https://github.com/advisories/GHSA-jfh8-c2jp-5v3q](https://github.com/advisories/GHSA-jfh8-c2jp-5v3q)

【漏洞描述】

Apache Log4j 是一款非常流行的开源的用于 Java 运行环境的日志记录工具包，大量的 Java 框架包括 Elasticsearch 的最新版本都使用了该组件，故影响范围非常之大。

近日, 随着 Apache Log4j 的远程代码执行最新漏洞细节被公开，攻击者可通过构造恶意请求利用该漏洞实现在目标服务器上执行任意代码。可导致服务器被黑客控制，从而进行页面篡改、数据窃取、挖矿、勒索等行为。建议使用该组件的用户第一时间启动应急响应进行修复。

简单总结一下就是，在使用 Log4j 打印输出的日志中，如果发现日志内容中包含关键词 `${`，那么这个里面包含的内容会当做变量来进行替换和执行，导致攻击者可以通过恶意构造日志内容来让 Java 进程来执行任意命令，达到攻击的效果。

【漏洞等级】：非常紧急

此次漏洞是用于 Log4j2 提供的 lookup 功能造成的，该功能允许开发者通过一些协议去读取相应环境中的配置。但在实现的过程中，并未对输入进行严格的判断，从而造成漏洞的发生。

【影响范围】：Java 类产品：Apache Log4j 2.x < 2.15.0-rc2

【攻击检测】

可以通过检查日志中是否存在 `jndi:ldap://`、`jndi:rmi` 等字符来发现可能的攻击行为。

## 处理办法

如果 Elasticsearch 不能修改配置、或者替换 Log4j 的 jar 包和重启集群的，可以使用极限网关来进行拦截或者参数替换甚至是直接阻断请求。
通过在网关层对发往 Elasticsearch 的请求统一进行参数检测，将包含的敏感关键词 `${` 进行替换或者直接拒绝，
可以防止带攻击的请求到达 Elasticsearch 服务端而被 Log4j 打印相关日志的时候执行恶意攻击命令，从而避免被攻击。

## 参考配置

下载最新的 `1.5.0-SNAPSHOT` 版本[http://release.elasticsearch.cn/gateway/snapshot/](http://release.elasticsearch.cn/gateway/snapshot/)

使用极限网关的 `context_filter` 过滤器，对请求上下文 `_ctx.request.to_string` 进行关键字检测，过滤掉恶意流量，从而阻断攻击。

```
path.data: data
path.logs: log

entry:
  - name: es_entrypoint
    enabled: true
    router: default
    max_concurrency: 20000
    network:
      binding: 0.0.0.0:8000

router:
  - name: default
    default_flow: main_flow

flow:
  - name: main_flow
    filter:
      - context_filter:
          context: _ctx.request.to_string
          action: redirect_flow
          status: 403
          flow: log4j_matched_flow
          must_not: # any match will be filtered
            regex:
              - \$\{.*?\}
              - "%24%7B.*?%7D" #urlencode
            contain:
              - "jndi:"
              - "jndi:ldap:"
              - "jndi:rmi:"
              - "jndi%3A" #urlencode
              - "jndi%3Aldap%3A" #urlencode
              - "jndi%3Armi%3A" #urlencode
      - elasticsearch:
          elasticsearch: es-server
  - name: log4j_matched_flow
    filter:
      - echo:
          message: 'Apache Log4j 2, Boom!'

elasticsearch:
  - name: es-server
    enabled: true
    endpoints:
      - http://localhost:9200
```

将测试命令 `${java:os}` 使用 urlencode 转码为 `%24%7Bjava%3Aos%7D`

不走网关：

```
~%  curl 'http://localhost:9200/index1/_search?q=%24%7Bjava%3Aos%7D'
{"error":{"root_cause":[{"type":"index_not_found_exception","reason":"no such index","resource.type":"index_or_alias","resource.id":"index1","index_uuid":"_na_","index":"index1"}],"type":"index_not_found_exception","reason":"no such index","resource.type":"index_or_alias","resource.id":"index1","index_uuid":"_na_","index":"index1"},"status":404}%
```

查看 Elasticsearch 端日志为：

```
[2021-12-11T01:49:50,303][DEBUG][r.suppressed             ] path: /index1/_search, params: {q=Mac OS X 10.13.4 unknown, architecture: x86_64-64, index=index1}
org.elasticsearch.index.IndexNotFoundException: no such index
	at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.infe(IndexNameExpressionResolver.java:678) ~[elasticsearch-5.6.15.jar:5.6.15]
	at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.innerResolve(IndexNameExpressionResolver.java:632) ~[elasticsearch-5.6.15.jar:5.6.15]
	at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.resolve(IndexNameExpressionResolver.java:580) ~[elasticsearch-5.6.15.jar:5.6.15]

```

可以看到查询条件里面的 `q=${java:os}` 被执行了，变成了 `q=Mac OS X 10.13.4 unknown, architecture: x86_64-64, index=index1`

走网关：

```
medcl@Medcl:~%  curl 'http://localhost:8000/index1/_search?q=%24%7Bjava%3Aos%7D'

Apache Log4j 2, Boom!%
```

可以看到请求被过滤到了。

其他命令可以试试：

```
#{java:vm}
~%  curl 'http://localhost:9200/index/_search?q=%24%7Bjava%3Avm%7D'
[2021-12-11T02:36:04,764][DEBUG][r.suppressed             ] [Medcl-2.local] path: /index/_search, params: {q=OpenJDK 64-Bit Server VM (build 25.72-b15, mixed mode), index=index}

~%  curl 'http://localhost:8000/index/_search?q=%24%7Bjava%3Avm%7D'
Apache Log4j 2, Boom!%

#{jndi:rmi://localhost:1099/api}
~%  curl 'http://localhost:9200/index/_search?q=%24%7Bjndi%3Armi%3A%2F%2Flocalhost%3A1099%2Fapi%7D'
2021-12-11 03:35:06,493 elasticsearch[YOmFJsW][search][T#3] ERROR An exception occurred processing Appender console java.lang.SecurityException: attempt to add a Permission to a readonly Permissions object

~%  curl 'http://localhost:8000/index/_search?q=%24%7Bjndi%3Armi%3A%2F%2Flocalhost%3A1099%2Fapi%7D'
Apache Log4j 2, Boom!%
```

> 使用极限网关处置类似安全事件的好处是，Elasticsearch 服务器不用做任何变动，尤其是大规模集群的场景，可以节省大量的工作，提升效率，非常灵活，缩短安全处置的时间，降低企业风险。
