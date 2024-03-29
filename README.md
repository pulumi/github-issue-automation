# github-issue-automation (DEPRECATED)

**NOTE:** As of January 17, 2024, this repository is DEPRECATED.
See https://github.com/pulumi/ci-mgmt/issues/724, where we discuss the pros and cons of using Zapier, and have opted for a poll-based system instead.
The production stack was deleted.

This repo contains code and infra to automate the creation of GitHub issues in Pulumi provider repos for upstream updates. 
These include:

1. Terraform provider updates that trigger the creation of issues in downstream bridged providers (the `new-release-handler` Lambda).

## new-release-handler

Because TF providers are not in repos Pulumi controls, we need to monitor RSS feeds for new releases (as opposed to releases of Pulumi providers, where we can use GHA workflows to trigger automation). 
RSS feeds are monitored by [Zapier](https://zapier.com/app). 
(At the time of writing, creation of "Zaps" in Zapier do not support any automation, and thus are configured manually in the Zapier UI.) 
Zapier is configured to invoke the Lambda contained in this repository when new items appear in the monitored RSS feeds.

For an example of incoming Zapier events, see `sample-events/`. 

Ideally, we'd like to rename this Lambda `tf-provider-release-handler`, but because it would require manually updating all resources in Zapier, this is not worth the trouble.

### Access to Zapier

[pulumi-internal] Ask for an invite to the Pulumi account from @pulumi/platform-providers or any engineering manager.

## Deploy the code for local development:

1. Ensure that an environment variable `GITHUB_TOKEN` is set to a value that allows the creation of GitHub issues for all repositories in the `pulumi` GitHub org and can add issues to the Platform Integrations Board.
1. Set your `AWS_PROFILE` environment variable to the pulumi-dev-sandbox account and log in.
1. `make deploy-dev`

Some resources in this stack have static names because they are referenced outside the stack by name, therefore the stack cannot be deployed in the same AWS account more than once. 
Be sure to tear down the stack if you deploy locally for testing to avoid causing issues for teammates.

## Testing changes for automatic upstream provider updates

1. In the AWS console, find your development Lambda, send a test event (see .`sample-events/sample-event.json` for an example and edit the fields as necessary) via the `Test` tab.
1. Observe the Pulumi provider action "Update upstream provider" being triggered. *Note*: this is a real Action and will result in an automatic update and merge if the Action passes.
1. When done testing, tear down the pulumi dev stack: `cd pulumi && pulumi destroy pulumi/dev`.

## Deployment

Main is automatically deployed on Pulumi's corp account with Pulumi Deployments.
