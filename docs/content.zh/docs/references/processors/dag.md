---
title: "dag"
---

# dag

## 描述

dag 处理器用来管理任务的并行调度。

## 配置示例

下面的这个例子，定义了一个名为 `racing_example` 的服务，`auto_start` 设置为自动启动，`processor` 设置依次执行的每个处理单元，其中 `dag` 处理器支持多个任务并行执行，支持 `wait_all` 和 `first_win` 两种聚合模式，如下：

```
pipeline:
  - name: racing_example
    auto_start: true
    processor:
    - echo: #ready, set, go
        message: read,set,go
    - dag:
        mode: wait_all #first_win, wait_all
        parallel:
          - echo: #player1
              message: player1
          - echo: #player2
              message: player2
          - echo: #player3
              message: player3
        end:
          - echo: #checking score
              message: checking score
          - echo: #announce champion
              message: 'announce champion'
    - echo: #done
        message: racing finished
```

上面的 `echo` 处理器非常简单，用来输出一个指定的消息，这个管道模拟的是一个赛跑的场景，palyer1、2、3 并行赛跑，全部跑完之后再进行算分和宣布比赛冠军，最后输出结束信息，程序运行输出如下：

```
[10-12 14:59:22] [INF] [echo.go:36] message:read,set,go
[10-12 14:59:22] [INF] [echo.go:36] message:player1
[10-12 14:59:22] [INF] [echo.go:36] message:player2
[10-12 14:59:22] [INF] [echo.go:36] message:player3
[10-12 14:59:22] [INF] [echo.go:36] message:checking score
[10-12 14:59:22] [INF] [echo.go:36] message:announce champion
[10-12 14:59:22] [INF] [echo.go:36] message:racing finished
```

## 参数说明

| 名称     | 类型   | 说明                                                                                                                                            |
| -------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| mode     | string | 任务结果的聚合模式，设置 `first_win` 表示并行里面的任意任务执行完就继续往下执行，而设置 `wait_all` 表示需要等待所有任务执行完毕才继续往后执行。 |
| parallel | array  | 任务数组列表，依次定义多个子任务                                                                                                                |
| end      | array  | 任务数组列表，并行任务之后再执行的任务                                                                                                          |
