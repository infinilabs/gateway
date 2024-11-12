---
title: "translog"
---

# translog

## Description

The translog filter is used to save received requests to local files and compress them. It can record some or complete request logs for archiving and request replay.

## Configuration Example

A simple example is as follows:

```
flow:
  - name: translog
    filter:
      - translog:
          max_file_age: 7
          max_file_count: 10
```

## Parameter Description

| Name                         | Type   | Description                                                                                                   |
| ---------------------------- | ------ | ------------------------------------------------------------------------------------------------------------- |
| path                         | string | Root directory for log storage, which is the `translog` subdirectory in the gateway data directory by default |
| category                     | string | Level-2 subdirectory for differentiating different logs, which is `default` by default.                       |
| filename                     | string | Name of the log storage file, which is `translog.log` by default.                                             |
| rotate.compress_after_rotate | bool   | Whether to compress and archive files after scrolling. The default value is `true`.                           |
| rotate.max_file_age          | int    | Maximum number of days that archived files can be retained, which is `30` days by default.                    |
| rotate.max_file_count        | int    | Maximum number of archived files that can be retained, which is `100` by default.                             |
| rotate.max_file_size_in_mb   | int    | Maximum size of a single archived file, in bytes. The default value is `1024` MB.                             |
