Discoverd Slack Notifier
========================

Send Slack notifications to `SLACK_WEBHOOK` when discoverd services go up or down.

Deploy
------

Clone this repository then run the following:

```
$ flynn create discoverd-slack-notifier
$ flynn env set SLACK_WEBHOOK="https://hooks.slack.com/services/XXXXXXXXX/XXXXXXXXX/XXXXXXXXXXXXXXXXXXXXXXXX"
$ flynn env set DISCOVERD_SERVICES="my-app-web,my-other-app-web"
$ git push flynn master
$ flynn scale notifier=1
```
