---
title: "在线查询修复的实现"
weight: 10
---

# 在线查询修复的实现

在某些情况下，您可能会碰到业务代码生成的 QueryDSL 存在不合理的情况，一般做法是需要修改业务代码并发布上线，
如果上线新版本需要很长的时间，比如没有到投产窗口，或者封网，又或者需要和其他的代码提交一起上线，往往意味着需要大量的测试，
而生产环境的故障要立马解决，客户不能等啊，怎么办？

别着急，您可以使用极限网关来对查询进行动态修复。

## 举个例子

比如下面的这个查询：

```
GET _search
{
 "size": 1000000
 , "explain": true
}
```

参数 `size` 设置的太大了，刚开始没有发现问题，随着数据越来越多，返回的数据太多势必会造成性能的急剧下降，
另外参数 `explain` 的开启也会造成不必要的性能开销，一般只在开发调试的时候才会用到这个功能。

通过在网关里面增加一个 `request_body_json_set` 过滤器，可以动态替换指定请求体 JSON PATH 的值，上面的例子对应的配置如下：

```
flow:
- name: rewrite_query
  filter:
    - request_body_json_set:
       path:
         - explain -> false
         - size -> 10
   - dump_request_body:
   - elasticsearch:
       elasticsearch: dev
```

通过重新设置 `explain` 和 `size` 参数，现在我们查询发给 Elasticsearch 前会被改写成如下格式：

```
{
 "size": 10, "explain": false
}
```

成功修复线上问题。

## 再举个例子

看下面的这个查询，编写代码的程序员写错了需要查询的字段名，应该是 `name`，但是写成了 `name1`，参数 `size` 也设置的特别大，如下：

```
GET medcl/_search
{
  "aggs": {
    "total_num": {
      "terms": {
        "field": "name1",
        "size": 1000000
      }
    }
  }
}
```

然后，系统居然上线了，这不查询就出问题了嘛。
哎，别着急，在网关请求流程里面增加如下过滤器配置就行了：

```
flow:
- name: rewrite_query
  filter:
    - request_body_json_set:
       path:
         - aggs.total_num.terms.field -> "name"
         - aggs.total_num.terms.size -> 10
         - size -> 0
   - dump_request_body:
   - elasticsearch:
       elasticsearch: dev
```

上面的配置，我们通过请求体 JSON 的路径直接替换了其数据，并且新增了一个参数来不返回查询文档，因为只需要聚合结果就行了。

## 再举个例子

用户的查询为：

```
{
  "query":{
	"bool":{
	   "should":[{"term":{"isDel":0}},{"match":{"type":"order"}}]
	}
}
}
```

现在希望将其中的 term 查询换成等价的 range 查询，即如下：

```
{
  "query":{
	"bool":{
	   "should":[{ "range": { "isDel": {"gte": 0,"lte": 0 }}},{"match":{"type":"order"}}]
	}
}
}
```

使用下面的配置即可：

```
flow:
  - name: rewrite_query
    filter:
      - request_body_json_del:
          path:
            - query.bool.should.[0]
      - request_body_json_set:
          path:
            - query.bool.should.[1].range.isDel.gte -> 0
            - query.bool.should.[1].range.isDel.lte -> 0
      - dump_request_body:
      - elasticsearch:
          elasticsearch: dev
```

上面的配置，首先使用了一个 `request_body_json_del` 来删除查询 should 里面的第一个元素，也就是要替换掉的 Term 子查询，
然后现在只剩一个 Match 查询了，现在增加一个 Should 的子查询，新增下标的注意应该为 `1`，分别设置 Range 查询的各个属性即可。

## 进一步完善

上面的例子都是直接替换查询，不过一般情况下，你可能还需要进行一个判断来决定是否进行替换，比如当
`_ctx.request.body_json.query.bool.should.[0].term.isDel` JSON 字段存在才进行替换，网关的[条件判断](../references/flow/#%E6%9D%A1%E4%BB%B6%E5%AE%9A%E4%B9%89)非常灵活如下，配置如下：

```
flow:
  - name: cache_first
    filter:
      - if:
          and:
            - exists: ['_ctx.request.body_json.query.bool.should.[0].term.isDel']
        then:
          - request_body_json_del:
              path:
                - query.bool.should.[0]
          - request_body_json_set:
              path:
                - query.bool.should.[1].range.isDel.gte -> 0
                - query.bool.should.[1].range.isDel.lte -> 0
          - dump_request_body:
      - elasticsearch:
          elasticsearch: dev
```

完美！
