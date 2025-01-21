---
title: "javascript"
---

# javascript

## 描述

javascript 过滤器可用于通过用 javascript 编写脚本来执行您自己的处理逻辑，从而提供最大的灵活性。

## 配置示例

一个简单的示例如下：

```
flow:
 - name: test
   filter:
    - javascript:
        source: >
          function process(ctx) {
            var console = require('console');
            console.log("hello from javascript");
          }
```

这个脚本里面的 `process` 是一个内置的函数，用来处理传进来的上下文信息，函数里面可以自定义业务逻辑。

如果脚本比较复杂，也支持通过文件的方式从加载：

```
flow:
 - name: test
   filter:
    - javascript:
        file: example.js
```

这里的 `example.js` 是文件的保存路径。

## 参数说明

| 名称   | 类型   | 描述                                                                                |
| ------ | ------ | ----------------------------------------------------------------------------------- |
| source | string | 要执行的 Javascript 代码。                                                          |
| file   | string | 要加载的脚本文件的路径。相对路径被解释为相对于网关实例数据目录的 `scripts` 子目录。 |
| params | map    | 一个参数字典，传递给脚本的 `register` 方法。                                        |

## 上下文 API

传递给处理方法的上下文对象具有以下 API 可以被使用。有关上下文的更多信息，请查看 [Request Context](../context/)。

| 方法                     | 描述                                                                                                                                                                                 |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Get(string)              | 从上下文中获取一个值。如果字段不存在，则返回 null。 eg: `var value = event.Get(key);`                                                                                                |
| Put(string, value)       | 在上下文中输入一个值。如果字段已经设置，则返回以前的值。如果字段存在但不是对象无法设置，则会抛出异常。 eg: `var old = event.Put(key, value);`                                        |
| Rename(string, string)   | 在上下文中重命名一个字段。目标键必须不存在。如果成功地将源键重命名为目标键，则返回 true。 eg: `var success = event.Rename("source", "target");`                                      |
| Delete(string)           | 从上下文中删除一个字段。成功时返回 true。 eg: `var deleted = event.Delete("user.email");`                                                                                            |
| Tag(string)              | 如果 Tag 不存在，则将 Tag 追加到 Tag 字段。如果 Tag 存在但不是字符串或字符串列表，则抛出异常。 eg: `event.Tag("user_event");`                                                        |
| AppendTo(string, string) | 一个专门的追加字段值的方法，它将现有值转换为数组，并在值不存在时追加该值。如果现有值不是字符串或字符串数组，则抛出异常。 eg: `event.AppendTo("error.message", "invalid file hash");` |

## 外部参数的使用

下面的例子，介绍了如何使用 `params` 来传递变量，脚本可以加载来自文件，方便复用程序脚本。

```
flow:
 - name: test
   filter:
    - javascript:
        params:
          keyword: [ "hello", "world", "scripts" ]
        source: >
          var console = require('console');
          var params = {keyword: []};
          function register(scriptParams) {
              params = scriptParams;
          }
          function process(ctx) {
            console.info("keyword comes from params: [%s]", params.keyword);
          }
```

`register` 是一个内置的函数，用来初始化外部参数。
