# github-issue-automation

This repo contains code and infra to automate the creation of GitHub issues for Terraform provider updates.  GitHub release RSS feeds are monitored by [Zapier](https://zapier.com/app).  (At the time of writing, creation of "Zaps" in Zapier do not support any automation, and thus are configured manually in the Zapier UI.)  Zapier is configured to invoke the Lambda contained in this repository when new items appear in the monitored RSS feeds.  For an example of incoming Zapier events, see `sample-events/`. 

## Deploy the code for local development:

1. Ensure that an environment variable `GITHUB_TOKEN` is set to a value that allows the creation of GitHub issues for all repositories in the `pulumi` GitHub org and can add issues to the Platform Integrations Board.
1. Set up any AWS environment variables
1. `make deploy`

## Testing changes for automatic upstream provider updates

1. In the AWS console, find your development Lambda, send a [test event](./sample-events/sample-event.json) via the `Test` tab, using an upstream provider release as the link.
2. Observe the Pulumi provider action "Update upstream provider" being triggered. *Note*: this is a real Action and will result in an automatic update and merge if the Action passes.
3. When done testing, tear down the pulumi dev stack: `cd pulumi && pulumi destroy pulumi/dev`.
