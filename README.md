# slack-jira-bot

Extremely limited JIRA bot for slack, that we use at [litl](https://litl.com/). It's
pretty specific to our setup, but you might find it useful as a starting point
to writing your own version.

## What it does

* It will output a brief description of a bug whenever someone mentions a key (PJT-42)
* It can resolve bugs for you with `jirabot: resolve PJT-42 Some nice comment`
* It can output a list of issues currently on the active sprint, with `jirabot: sprint`,
optionally filtered by Component with `jirabot: sprint <some component>`

## How to use

You need to setup a bot for your team. We have been using a
[custom bot user](https://my.slack.com/services/new/bot).
After configuring it you will get an API token, set it as the `SLACK_JIRA_BOT_SLACK_TOKEN` environment variable.

You will also need a JIRA user to represent the bot. Configure the `SLACK_JIRA_BOT_JIRA_USERNAME`
and `SLACK_JIRA_BOT_JIRA_PASSWORD` environment variables accordingly.

Finally, configure `SLACK_JIRA_BOT_JIRA_URL` to the base URL of your JIRA installation, `SLACK_JIRA_BOT_PROJECT_PREFIX` to the
prefix of your JIRA project, and `SLACK_JIRA_BOT_JIRA_AGILE_BOARD` to the ID of the JIRA Agile (ex-greenhopper) scrum board.

All in all, something like:


```
$ SLACK_JIRA_BOT_JIRA_URL="https://myjira/" \
  SLACK_JIRA_BOT_JIRA_USERNAME="jirabot" \
  SLACK_JIRA_BOT_JIRA_PASSWORD="jirabotpassword" \
  SLACK_JIRA_BOT_JIRA_AGILE_BOARD=30 \
  SLACK_JIRA_BOT_JIRA_PROJECT_PREFIX="PJT" \
  SLACK_JIRA_BOT_SLACK_TOKEN="xoxb-XXXXXXXXXXX-YYYYYYYYYYYYYYYY" \
  slack-jira-bot
```
