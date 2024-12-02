---
title: "dag"
---

# dag

## Description

The dag processor is used to manage the concurrent scheduling of tasks.

## Configuration Example

The following example defines a service named `racing_example` and `auto_start` is set to true. Processing units to be executed in sequence are set in `processor`, the `dag` processor supports concurrent execution of multiple tasks and the `wait_all` and `first_win` aggregation modes.

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

The `echo` processor above is very simple and is used to output a specified message. This pipeline simulates a race scene, in which players 1, 2, and 3 run at the same time. After they run, the scores are calculated and the winner is announced, and finally the completion information is output. The output of the program is as follows:

```
[10-12 14:59:22] [INF] [echo.go:36] message:read,set,go
[10-12 14:59:22] [INF] [echo.go:36] message:player1
[10-12 14:59:22] [INF] [echo.go:36] message:player2
[10-12 14:59:22] [INF] [echo.go:36] message:player3
[10-12 14:59:22] [INF] [echo.go:36] message:checking score
[10-12 14:59:22] [INF] [echo.go:36] message:announce champion
[10-12 14:59:22] [INF] [echo.go:36] message:racing finished
```

## Parameter Description

| Name     | Type   | Description                                                                                                                                                                                                                                                                                   |
| -------- | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| mode     | string | Aggregation mode of task results. The value `first_win` indicates that the program continues further execution after any of the concurrent tasks is completed, and the value `wait_all` indicates that the program continues further execution only after all concurrent tasks are completed. |
| parallel | array  | Task array list, in which multiple subtasks are defined in sequence.                                                                                                                                                                                                                          |
| end      | array  | Task array list, which lists tasks to be executed after concurrent tasks are completed.                                                                                                                                                                                                       |
