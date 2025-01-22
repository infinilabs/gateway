---
title: "bulk_request_mutate"
---

# bulk_request_mutate

## 描述

bulk_request_mutate 过滤器用来干预 Elasticsearch 的 Bulk 请求。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: bulk_request_mutate
    filter:
      - bulk_request_mutate:
          fix_null_id: true
          generate_enhanced_id: true
#          fix_null_type: true
#          default_type: m-type
#          default_index: m-index
#          index_rename:
#            "*": index-new
#            index1: index-new
#            index2: index-new
#            index3: index3-new
#            index4: index3-new
#            medcl-dr3: index3-new
#          type_rename:
#            "*": type-new
#            type1: type-new
#            type2: type-new
#            doc: type-new
#            doc1: type-new

...
```

## 参数说明

| 名称                 | 类型   | 说明                                                                        |
| -------------------- | ------ | --------------------------------------------------------------------------- |
| fix_null_type        | bool   | 是否修复不带 `_type` 的请求，和参数 `default_type` 配合使用                 |
| fix_null_id          | bool   | 是否修复不带 `_id` 的请求，生成一个随机 id，如 `c616rhkgq9s7q1h89ig0`       |
| remove_type          | bool   | 是否移除 `_type` 参数，Elasticsearch 8.0 之后不支持 `_type` 参数            |
| generate_enhanced_id | bool   | 是否生成一个增强的 id 类型，如 `c616rhkgq9s7q1h89ig0-1635937734071093-10`   |
| default_index        | string | 默认的索引名称，如果元数据里面没有指定，则使用该默认值                      |
| default_type         | string | 默认的文档 type，如果没有元数据里面没有指定，则使用该默认值                 |
| index_rename         | map    | 将索引名称进行重命名，支持 `*` 来覆盖所有的索引名称                         |
| type_rename          | map    | 将 type 进行重命名，支持 `*` 来覆盖所有的 type 名称                         |
| pipeline             | string | 指定 bulk 请求的 `pipeline` 参数                                            |
| remove_pipeline      | bool   | 是否移除 bulk 请求中的 `pipeline` 参数                                      |
| safety_parse         | bool   | 是否采用安全的 bulk 元数据解析方法，默认 `true`                             |
| doc_buffer_size      | int    | 当采用不安全的 bulk 元数据解析方法时，使用的 buffer 大小，默认 `256 * 1024` |
