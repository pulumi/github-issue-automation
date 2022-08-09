package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/google/go-github/v41/github"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/oauth2"
	"log"
	"os"
	"time"
)

type releaseEvent struct {
	Repo    string `json:"repo"`
	TagName string `json:"tag_name"`
	Url     string `json:"url"`
}

func main() {
	lambda.Start(LambdaHandler)
}

func LambdaHandler(event releaseEvent) error {
	bytes, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshall incoming event: %s", err)
	}

	log.Printf("Event = %s", bytes)

	if event.Repo == "" {
		return errors.New("the 'repo' field is required on the incoming event")
	}

	if event.TagName == "" {
		return errors.New("the 'tagName' field is required on the incoming event")
	}

	if err != nil {
		return err
	}

	downstreamRepos := getDownstreamRepos(event.Repo)

	if len(downstreamRepos) == 0 {
		log.Printf("Incoming event repo '%s' has no downstream repos configured. Nothing to do. Exiting.", event.Repo)
		return nil
	}

	ctx := context.Background()
	gitHubToken, err := getGitHubToken()
	gitHubClient := newGitHubClient(ctx, gitHubToken)

	var result *multierror.Error
	for _, repo := range downstreamRepos {
		err = createIssue(repo, event, ctx, gitHubClient)
		if err != nil {
			log.Printf("Error while creating issue for repo '%s'", repo)
			result = multierror.Append(result, err)
		} else {
			log.Printf("Successfully created issue for repo '%s", repo)
		}
	}

	if result.ErrorOrNil() != nil {
		log.Printf("Encountered at least 1 error creating issues. Exiting.")
		return result
	}

	log.Print("Done.")

	return nil
}

func getGitHubToken() (string, error) {
	arn := os.Getenv("GITHUB_TOKEN_SECRET_ARN")
	if arn == "" {
		panic("The environment variable 'GITHUB_TOKEN_SECRET_ARN' must be set.")
	}

	newSession, err := session.NewSession()
	if err != nil {
		return "", err
	}

	client := secretsmanager.New(newSession)
	secret, err := client.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(arn),
	})
	if err != nil {
		return "", err
	}

	return *secret.SecretString, nil
}

func newGitHubClient(ctx context.Context, gitHubToken string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: gitHubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	return client
}

func getIssues(ctx context.Context, client *github.Client, repository string) ([]*github.Issue, error) {
	var allIssues []*github.Issue

	opts := &github.IssueListByRepoOptions{
		Since: time.Now().AddDate(0, 0, -30),
		State: "all",
	}

	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, "pulumi", repository, opts)
		if err != nil {
			return nil, err
		}

		allIssues = append(allIssues, issues...)

		if resp.NextPage == 0 {
			break
		}

		opts.Page = resp.NextPage
	}

	return allIssues, nil
}

// createIssue creates an issue on the project board and sets it to the Provider Upgrades column
func createIssue(repoName string, event releaseEvent, ctx context.Context, gitHubClient *github.Client) error {
	issueTitle := fmt.Sprintf("Upgrade %s to %s", event.Repo, event.TagName)

	log.Printf("Checking for an existing issue in repo '%s'", repoName)
	issues, err := getIssues(ctx, gitHubClient, repoName)
	if err != nil {
		return err
	}

	issueExists := false
	for _, issue := range issues {
		if *issue.Title == issueTitle {
			issueExists = true
			break
		}
	}

	if issueExists {
		log.Printf("There is already an issue with the title '%s' in repo 'pulumi/%s'.", issueTitle, repoName)
		return nil
	} else {
		log.Print("Did not find an existing issue.  Creating a new issue.")

		body := fmt.Sprintf("Release details: %s", event.Url)

		issue, _, err := gitHubClient.Issues.Create(ctx, "pulumi", repoName, &github.IssueRequest{
			Title:  github.String(issueTitle),
			Labels: &[]string{"kind/enhancement"},
			Body:   github.String(body),
		})
		if err != nil {
			return err
		}

		log.Printf("Adding issue to project board.")
		//platformIntegrationsBoardId := 12058265
		providerUpgradesColumnsId := int64(14558007)
		_, _, err = gitHubClient.Projects.CreateProjectCard(ctx, providerUpgradesColumnsId, &github.ProjectCardOptions{
			ContentID: *issue.ID,
			// Not documented in the API.  See instead: https://stackoverflow.com/questions/57024087/github-api-how-to-move-an-issue-to-a-project
			ContentType: "Issue",
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func getDownstreamRepos(upstreamRepoName string) []string {
	switch upstreamRepoName {
	case "fake-aws":
		return []string{"fake-awsx"}
	case "pulumi-aws":
		return []string{"pulumi-awsx", "pulumi-eks", "pulumi-aws-apigateway"}
	default:
		return []string{}
	}
}
