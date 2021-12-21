# github-issue-automation

This repo contains code and infra to automate the creation of GitHub issues for Terraform provider updates.  GitHub release RSS feeds are monitored by [Zapier](https://zapier.com/app).  (At the time of writing, creation of "Zaps" in Zapier do not support any automation, and thus are configured manually in the Zapier UI.)  Zapier is configured to invoke the Lambda contained in this repository when new items appear in the monitored RSS feeds.  For an example of incoming Zapier events, see `sample-events/`. 

To deploy the code:

1. Ensure that an environment variable `GITHUB_TOKEN` is set to a value that allows the creation of GitHub issues for all repositories in the `pulumi` GitHub org and can add issues to the Platform Integrations Board.
1. `make deploy`
