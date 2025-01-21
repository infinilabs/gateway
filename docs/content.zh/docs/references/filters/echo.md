---
title: "echo"
asciinema: true
weight: 1
---

# echo

## 描述

echo 过滤器是一个用于在返回结果里面输出指定字符信息的过滤器，常用于调试。

## 功能演示

{{< asciinema key="/echo_helloworld" speed="2"  autoplay="1"  rows="30" preload="1" >}}

## 配置示例

一个简单的示例如下：

```
flow:
  - name: hello_world
    filter:
      - echo:
          message: "hello infini\n"
```

echo 过滤器可以设置重复输出相同的字符的次数，示例如下：

```
...
   - echo:
       message: "hello gateway\n"
       repeat: 3
...
```

## 参数说明

| 名称          | 类型     | 说明                                    |
| ------------- | -------- | --------------------------------------- |
| message       | string   | 需要输出的字符内容，默认 `.`            |
| messages      | []string | 需要输出的字符内容列表                  |
| status        | int      | HTTP 状态码，默认 `200`                 |
| repeat        | int      | 重复次数                                |
| continue      | bool     | 是否继续后续流程，默认为 `true`         |
| response      | bool     | 是否在 HTTP 返回输出，默认为 `true`     |
| stdout        | bool     | 是否在终端也打印输出，默认为 `false`    |
| logging       | bool     | 是否输出为日志数据，默认为 `false`      |
| logging_level | string   | 输出为日志数据的日志级别，默认为 `info` |
