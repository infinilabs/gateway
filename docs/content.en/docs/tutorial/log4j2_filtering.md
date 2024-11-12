---
title: "Protect Elasticsearch from Apache Log4j Vulnerability"
weight: 10
---

# Protect Elasticsearch from Apache Log4j Vulnerability

CVE Address

[https://github.com/advisories/GHSA-jfh8-c2jp-5v3q](https://github.com/advisories/GHSA-jfh8-c2jp-5v3q)

Vulnerability Description

Apache Log4j is a very popular open source logging toolkit used for the Java runtime environment. Many Java frameworks including Elasticsearch of the latest version, use this component. Therefore, the scope of impact is huge.

The latest vulnerability existing in the execution of Apache Log4j's remote code was revealed recently. Attackers can construct malicious requests and utilize this vulnerability to execute arbitrary code on a target server. As a result, the server can be controlled by hackers, who can then conduct page tampering, data theft, mining, extortion, and other behaviors. Users who use this component are advised to immediately initiate emergency response for fixing.

Basically, if a log output by Log4j contains the keyword `${`, the log is replaced as a variable and then the variable operation is executed. Attackers can maliciously construct log content to make Java processes to execute arbitrary commands, achieving the attack purpose.

Vulnerability Level: very urgent

The vulnerability is caused by the lookup function provided by Log4j2. This function allows developers to read configurations in the environment by using a number of protocols. However, the input is not strictly judged in the implementation, resulting in the vulnerability.

Impact Scope: Java products: Apache Log4j 2.x < 2.15.0-rc2

Attack Detection

You can check logs for `jndi:ldap://`, `jndi:rmi`, and other characters to find out possible attacks.

## Handling Method

If Elasticsearch does not support configuration modification, Jar package replacement of Log4j, or cluster restart, you can use INFINI Gateway to intercept requests, replace parameters, and even directly block requests.
You can use INFINI Gateway to check parameters in requests sent to Elasticsearch and replace or reject content that contains the sensitive keyword `${`.
In this way, INFINI Gateway can prevent the execution of malicious attack commands during Log4j logging after attack-contained requests are sent to Elasticsearch, thereby preventing attacks.

## Reference Configuration

Download the latest `1.5.0-SNAPSHOT` version: [http://release.elasticsearch.cn/gateway/snapshot/](http://release.elasticsearch.cn/gateway/snapshot/)

The `context_filter` filter of INFINI Gateway can be used to detect the keywords of the request context `_ctx.request.to_string` and filter out malicious traffic, thereby blocking attacks.

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

Use urlencode to convert the test command `${java:os}` into `%24%7Bjava%3Aos%7D`.

Request calling execution result when requests do not need to pass through the gateway:

```
~%  curl 'http://localhost:9200/index1/_search?q=%24%7Bjava%3Aos%7D'
{"error":{"root_cause":[{"type":"index_not_found_exception","reason":"no such index","resource.type":"index_or_alias","resource.id":"index1","index_uuid":"_na_","index":"index1"}],"type":"index_not_found_exception","reason":"no such index","resource.type":"index_or_alias","resource.id":"index1","index_uuid":"_na_","index":"index1"},"status":404}%
```

Logs on Elasticsearch are as follows:

```
[2021-12-11T01:49:50,303][DEBUG][r.suppressed             ] path: /index1/_search, params: {q=Mac OS X 10.13.4 unknown, architecture: x86_64-64, index=index1}
org.elasticsearch.index.IndexNotFoundException: no such index
	at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.infe(IndexNameExpressionResolver.java:678) ~[elasticsearch-5.6.15.jar:5.6.15]
	at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.innerResolve(IndexNameExpressionResolver.java:632) ~[elasticsearch-5.6.15.jar:5.6.15]
	at org.elasticsearch.cluster.metadata.IndexNameExpressionResolver$WildcardExpressionResolver.resolve(IndexNameExpressionResolver.java:580) ~[elasticsearch-5.6.15.jar:5.6.15]

```

The logs above show that `q=${java:os}` in query conditions is executed and is changed to `q=Mac OS X 10.13.4 unknown, architecture: x86_64-64, index=index1`.

Request calling execution result when requests need to pass through the gateway:

```
medcl@Medcl:~%  curl 'http://localhost:8000/index1/_search?q=%24%7Bjava%3Aos%7D'

Apache Log4j 2, Boom!%
```

The logs above show that requests are filtered out.

You can try other commands to check whether malicious requests are intercepted:

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

> The benefits of using INFINI Gateway is that no change needs to be made to the Elasticsearch server, especially in large-scale cluster scenarios. The flexible INFINI Gateway can significantly reduce workload, improve efficiency, shorten the security processing time, and reduce enterprise risks.
