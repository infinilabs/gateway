---
title: "bulk_request_mutate"
---

# bulk_request_mutate

## Description

The bulk_request_mutate filter is used to mutate bulk requests of Elasticsearch.

## Configuration Example

A simple example is as follows:

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

## Parameter Description

| Name                 | Type   | Description                                                                                                           |
| -------------------- | ------ | --------------------------------------------------------------------------------------------------------------------- |
| fix_null_type        | bool   | Whether to fix a request that does not carry `_type`. It is used in collaboration with the `default_type` parameter.  |
| fix_null_id          | bool   | Whether to fix a request that does not carry `_id` and generate a random ID, for example, `c616rhkgq9s7q1h89ig0`      |
| remove_type          | bool   | Whether to remove the `_type` parameter. Elasticsearch versions higher than 8.0 do not support the `_type` parameter. |
| generate_enhanced_id | bool   | Whether to generate an enhanced ID, such as `c616rhkgq9s7q1h89ig0-1635937734071093-10`.                               |
| default_index        | string | Default index name, which is used if no index name is specified in metadata                                           |
| default_type         | string | Default document type, which is used if no document type is specified in metadata                                     |
| index_rename         | map    | Index name used for renaming. You can use `*` to overwrite all index names.                                           |
| type_rename          | map    | Type used for renaming. You can use `*` to overwrite all type names.                                                  |
| pipeline             | string | `pipeline` parameter of a specified bulk request                                                                      |
| remove_pipeline      | bool   | Whether to remove the `pipeline` parameter from the bulk request                                                      |
| safety_parse         | bool   | Whether to use a secure bulk metadata parsing method. The default value is `true`.                                    |
| doc_buffer_size      | int    | Buffer size when an insecure bulk metadata parsing method is adopted. The default value is `256 * 1024`.              |
