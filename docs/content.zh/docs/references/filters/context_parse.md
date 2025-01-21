---
title: "context_parse"
---

# context_parse

## 描述

context_parse 过滤器用来对上下文变量进行字段的提取，并存放到上下文中。

## 配置示例

一个简单的示例如下：

```
flow:
  - name: context_parse
    filter:
      - context_parse:
          context: _ctx.request.path
          pattern: ^\/.*?\d{4}\.(?P<month>\d{2})\.(?P<day>\d{2}).*?
          group: "parsed_index"
```

通过 `context_parse` 可以提取请求如：`/abd-2023.02.06-abc/_search`，得到新的上下文变量 `parsed_index.month` 和 `parsed_index.day`。

## 参数说明

| 名称       | 类型   | 说明                                     |
| ---------- | ------ | ---------------------------------------- |
| context    | string | 上下文变量名称                           |
| pattern    | string | 用来提取字段的正则表达式                 |
| skip_error | bool   | 是否忽略错误直接返回，如上下文变量不存在 |
| group      | string | 提取的字段是否存放到一个单独的分组下面   |
