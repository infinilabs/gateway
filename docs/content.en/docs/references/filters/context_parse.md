---
title: "context_parse"
---

# context_parse

## Description

context_parse filter is used to extract fields from context variables and store them in the context。

## Configuration Example

A simple example is as follows:

```
flow:
  - name: context_parse
    filter:
      - context_parse:
          context: _ctx.request.path
          pattern: ^\/.*?\d{4}\.(?P<month>\d{2})\.(?P<day>\d{2}).*?
          group: "parsed_index"
```

In above flow, the `context_parse` can extract fields from request：`/abd-2023.02.06-abc/_search`，get two new fields: `parsed_index.month` and `parsed_index.day`。

## Parameter Description

| Name       | Type   | Description                                                                                      |
| ---------- | ------ | ------------------------------------------------------------------------------------------------ |
| context    | string | Context variable                                                                                 |
| pattern    | string | The regular expression used to extract the field                                                 |
| skip_error | bool   | Whether to ignore the error and returned directly, such like the context variable does not exist |
| group      | string | Set the group name, which the extracted fields can be placed under a separate group              |
