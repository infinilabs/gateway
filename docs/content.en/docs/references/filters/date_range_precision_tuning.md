---
title: "date_range_precision_tuning"
---

# date_range_precision_tuning

## Description

The date_range_precision_tuning filter is used to reset the time precision for time range query. After the precision is adjusted, adjacent repeated requests initiated within a short period of time can be easily cached. For scenarios with low time precision but a large amount of data, for example, if Kibana is used for report analysis, you can reduce the precision to cache repeated query requests to reduce the pressure of the back-end server and accelerate the front-end report presentation.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: test
    filter:
      - date_range_precision_tuning:
          time_precision: 4
      - get_cache:
      - elasticsearch:
          elasticsearch: dev
      - set_cache:
```

## Precision Description

Queries sent by Kibana to Elasticsearch use the current time (Now) by default, which is accurate to milliseconds. You can set different precision levels to rewrite queries. See the following query example:

```
{"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.152Z","lte":"2020-09-26T08:21:12.152Z","format":"strict_date_optional_time"}
```

Set different precision levels. The query results after rewriting are as follows:

| Precision | New Query                                                                                                                       |
| --------- | ------------------------------------------------------------------------------------------------------------------------------- |
| 0         | {"range":{"@timestamp":{"gte":"2019-09-26T00:00:00.000Z","lte":"2020-09-26T23:59:59.999Z","format":"strict_date_optional_time"} |
| 1         | {"range":{"@timestamp":{"gte":"2019-09-26T00:00:00.000Z","lte":"2020-09-26T09:59:59.999Z","format":"strict_date_optional_time"} |
| 2         | {"range":{"@timestamp":{"gte":"2019-09-26T08:00:00.000Z","lte":"2020-09-26T08:59:59.999Z","format":"strict_date_optional_time"} |
| 3         | {"range":{"@timestamp":{"gte":"2019-09-26T08:20:00.000Z","lte":"2020-09-26T08:29:59.999Z","format":"strict_date_optional_time"} |
| 4         | {"range":{"@timestamp":{"gte":"2019-09-26T08:21:00.000Z","lte":"2020-09-26T08:21:59.999Z","format":"strict_date_optional_time"} |
| 5         | {"range":{"@timestamp":{"gte":"2019-09-26T08:21:10.000Z","lte":"2020-09-26T08:21:19.999Z","format":"strict_date_optional_time"} |
| 6         | {"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.000Z","lte":"2020-09-26T08:21:12.999Z","format":"strict_date_optional_time"} |
| 7         | {"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.100Z","lte":"2020-09-26T08:21:12.199Z","format":"strict_date_optional_time"} |
| 8         | {"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.150Z","lte":"2020-09-26T08:21:12.159Z","format":"strict_date_optional_time"} |
| 9         | {"range":{"@timestamp":{"gte":"2019-09-26T08:21:12.152Z","lte":"2020-09-26T08:21:12.152Z","format":"strict_date_optional_time"} |

## Parameter Description

| Name           | Type  | Description                                                                                                                                                                                                |
| -------------- | ----- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| time_precision | int   | Precision length of time, that is, the digit length of displayed time. The default value is `4` and the valid range is from 0 to 9.                                                                        |
| path_keywords  | array | Keyword contained in a request. The time precision is reset only for requests that contain the keywords, to prevent parsing of unnecessary requests. The default values are `_search` and `_async_search`. |
