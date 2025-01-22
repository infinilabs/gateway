---
title: "smtp"
---

# smtp

## 描述

smtp 处理器用来发送邮件，支持普通的文本邮件和 HTML 邮件，支持模版变量，支持附件嵌入到邮件正文，邮件的消息来自上下文。

## 配置示例

一个简单的示例如下：

```
pipeline:
  - name: send_email
    auto_start: true
    keep_running: true
    retry_delay_in_ms: 5000
    processor:
      - consumer:
          consumer:
            fetch_max_messages: 1
          max_worker_size: 200
          num_of_slices: 1
          idle_timeout_in_seconds: 30
          queue_selector:
            keys:
              - email_messages
          processor:
            - smtp:
                idle_timeout_in_seconds: 1
                server:
                  host: "smtp.ym.163.com"
                  port: 994
                  tls:  true
                auth:
                  username: "notify-test@infini.ltd"
                  password: "xxx"
                sender: "notify-test@infini.ltd"
                recipients:
                #                  to: ["Test <medcl@infini.ltd>"]
                #                  cc: ["INFINI Labs <hello@infini.ltd>"]
                variables: #default variables, can be used in templates
                  license_code: "N/A"
                templates:
                  trial_license:
                    subject: "$[[name]] 您好，请查收您的免费授权信息! [INFINI Labs]"
                    #                    content_type: 'text/plain'
                    #                    body: "$[[name]] 您好，请查收您的免费授权信息! [INFINI Labs]"
                    content_type: 'text/html'
                    body_file: '/Users/medcl/go/src/infini.sh/ops/assets/email_templates/send_trial_license.html'
#                    attachments: #use cid in html: <img width=100 height=100 id="1" src="cid:myimg1">
#                      - file: '/Users/medcl/Desktop/WechatIMG2783.png'
#                        content_type: 'image/png'
#                        cid: 'myimg1'
```


## 消息格式

SMTP 过滤器从上下文获取需要发送的邮件信息，如发给谁，使用哪个邮件模版，给邮件模版的变量参数等，消息格式为固定的，结果如下：

```
{
  "template": "trial_license",
  "email":["medcl@example.com"],
  "variables": {
    "name": "Medcl",
    "company": "INFINI Labs",
    "phone": "400-139-9200"
  }
}
```

字段 `template` 代表使用配置里面的模版，`email` 表示邮件的收件人信息，`variables` 定义了在模版里面需要用到的变量信息。



## 参数说明

| 名称     | 类型   | 说明                                   |
| -------- | ------ | -------------------------------------- |
| dial_timeout_in_seconds | int |  发送邮件的超时时间设置                 |
| server.host   | string | 邮件服务器地址       |
| server.port   | int | 邮件服务器端口       |
| tls     | bool |  是否开启 TLS 传输加密 |
| auth.username     | string |  邮件服务器访问身份 |
| auth.password     | string |  邮件服务器访问密码 |
| sender     | string |  发送人，默认和 `auth.username` 保持一致 |
| recipients.to     | array |  收件人，选填 |
| recipients.cc     | array |  抄送人，选填 |
| recipients.bcc     | array |  密送人，选填 |
| templates[NAME].content_type     | string |  邮件类型，`text/plain` 或者 `text/html`|
| templates[NAME].subject     | string |  邮件主题，支持模版变量|
| templates[NAME].body     | string |  邮件正文，支持模版变量|
| templates[NAME].body_file     | string |  来自文件的邮件正文，支持模版变量|
| templates[NAME].attachments[i].cid     | string |  附件CID，可以在正文中引用，如：`<img width=100 height=100 id="1" src="cid:myimg1">` |
| templates[NAME].attachments[i].file     | string |  附件文件路径|
| templates[NAME].attachments[i].content_type     | string |  附件文件类型，参考：http://en.wikipedia.org/wiki/MIME |
| message_field | string |  变量来自的上下文字段，默认 `message_field`                 |
| variable_start_tag | string |  变量 Tag 前缀，默认 `$[[`                 |
| variable_end_tag | string |  变量 Tag 后缀，默认 `]]`                 |
| variables |  array |  内置变量， 可被上下文变量覆盖                 |

