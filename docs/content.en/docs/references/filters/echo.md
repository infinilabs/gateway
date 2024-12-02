---
title: "echo"
asciinema: true
weight: 1
---

# echo

## Description

The echo filter is used to output specified characters in the returned result. It is often used for debugging.

## Function Demonstration

{{< asciinema key="/echo_helloworld" speed="2"  autoplay="1"  rows="30" preload="1" >}}

## Configuration Example

A simple example is as follows:

```
flow:
  - name: hello_world
    filter:
      - echo:
          message: "hello infini\n"
```

The echo filter allows you to set the number of times that same characters can be output repeatedly. See the following example.

```
...
   - echo:
       message: "hello gateway\n"
       repeat: 3
...
```

## Parameter Description

| Name          | Type     | Description                                                                     |
| ------------- | -------- | ------------------------------------------------------------------------------- |
| message       | string   | Characters to be output，default `.`                                            |
| messages      | []string | Characters list to be output                                                    |
| status        | int      | HTTP Status，default `200`                                                      |
| repeat        | int      | Number of repetition times                                                      |
| continue      | bool     | Whether to continue further filters，default `true`                             |
| response      | bool     | Whether to output to HTTP response，default `true`                              |
| stdout        | bool     | Whether the terminal also outputs the characters. The default value is `false`. |
| logging       | bool     | Whether to output as logging，default `false`                                   |
| logging_level | string   | The logging level for output logs，default `info`                               |
