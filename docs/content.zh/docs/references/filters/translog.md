---
title: "translog"
---

# translog

## 描述

translog 过滤器用来将收到的请求保存到本地文件，并压缩存放，可记录部分或完整的请求日志，用于归档和请求重放。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: translog
    filter:
      - translog:
          max_file_age: 7
          max_file_count: 10
```

## 参数说明

| 名称                         | 类型   | 说明                                                     |
| ---------------------------- | ------ | -------------------------------------------------------- |
| path                         | string | 日志存放根目录，默认为网关数据目录下的 `translog` 子目录 |
| category                     | string | 区分不同日志的二级分类子目录，默认为 `default`           |
| filename                     | string | 设置日志的文件名，默认为 `translog.log`                  |
| rotate.compress_after_rotate | bool   | 文件滚动之后是否压缩归档，默认为 `true`                  |
| rotate.max_file_age          | int    | 最多保留的归档文件天数，默认为 `30` 天                   |
| rotate.max_file_count        | int    | 最多保留的归档文件个数，默认为 `100` 个                  |
| rotate.max_file_size_in_mb   | int    | 单个归档文件的最大字节数，默认为 `1024` MB               |
