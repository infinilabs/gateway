---
title: "smtp"
---

# smtp

## Description

The SMTP processor is used to send emails, supporting both plain text and HTML emails. It supports template variables and allows attachments to be embedded in the email body. The email message is comes from the pipeline context.

## Configuration Example

A simple example is as follows:

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


## Message Format

The SMTP filter retrieves email information to be sent from the context, such as the recipient, which email template to use, and the variable parameters for the email template. The message format is fixed, and the result is as follows:

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

The field template represents the template to be used from the configuration. email represents the recipient information of the email. variables defines the variable information to be used in the template.


## Parameter Description

| Name                       | Type   | Description                                            |
| -------------------------- | ------ | ------------------------------------------------------ |
| dial_timeout_in_seconds    | int    | Timeout duration for sending emails                    |
| server.host                | string | Email server address                                   |
| server.port                | int    | Email server port                                      |
| tls                        | bool   | Whether to enable TLS encryption for transmission      |
| auth.username              | string | Email server access username                           |
| auth.password              | string | Email server access password                           |
| sender                     | string | Sender's email address (defaults to `auth.username`)    |
| recipients.to              | array  | Recipients' email addresses (optional)                 |
| recipients.cc              | array  | CC recipients' email addresses (optional)              |
| recipients.bcc             | array  | BCC recipients' email addresses (optional)             |
| templates[NAME].content_type | string | Email type, either `text/plain` or `text/html`         |
| templates[NAME].subject     | string | Email subject, supports template variables             |
| templates[NAME].body        | string | Email body, supports template variables                |
| templates[NAME].body_file   | string | Email body from a file, supports template variables    |
| templates[NAME].attachments[i].cid            | string | Attachment CID for referencing in the body             |
| templates[NAME].attachments[i].file           | string | Attachment file path                                  |
| templates[NAME].attachments[i].content_type   | string | Attachment file type, see: http://en.wikipedia.org/wiki/MIME |
| message_field              | string | Source field for variables (defaults to `message_field`) |
| variable_start_tag         | string | Variable tag prefix (defaults to `$[[`)                |
| variable_end_tag           | string | Variable tag suffix (defaults to `]]`)                 |
| variables                  | array  | Built-in variables that can be overridden by context variables |      |

