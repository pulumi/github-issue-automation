package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-lambda-go/lambda"
	"golang.org/x/oauth2"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/google/go-github/v41/github"
)

type NewRelease struct {
	Title string `json:"Title"`
	Link  string `json:"Link"`
	// Tracks the name of the Zap that originated the request:
	ZapierSource string `json:"ZapierSource"`
}

func main() {
	lambda.Start(LambdaHandler)
}

func LambdaHandler(event NewRelease) error {
	bytes, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshall incoming event: %s", err)
	}

	if event.Title == "" {
		return errors.New("the 'Title' field is required on the incoming event")
	}

	if event.Link == "" {
		return errors.New("the 'Link' field is required on the incoming event")
	}

	log.Printf("Event = %s", bytes)

	gitHubToken, err := getGitHubToken()
	if err != nil {
		return err
	}

	ctx := context.Background()
	gitHubClient := newGitHubClient(ctx, gitHubToken)

	pulumiRepo, err := parsePulumiRepo(event.Link)
	if err != nil {
		return err
	}

	tfRepo, err := parseTerraformRepo(event.Link)
	if err != nil {
		return err
	}

	version, err := parseVersion(event.Link)
	if err != nil {
		return err
	}

	if isPreRelease(version) {
		log.Printf("Version %s is pre-release. Nothing to do. Exiting.", version)
		return nil
	}

	issueTitle := fmt.Sprintf("Upgrade %s to %s", tfRepo, version)

	log.Printf("Checking for an existing issue in repo '%s'", pulumiRepo)
	issues, err := getIssues(ctx, gitHubClient, pulumiRepo)
	if err != nil {
		return err
	}

	issueExists := false
	var issueNumber int
	for _, issue := range issues {
		if *issue.Title == issueTitle {
			issueNumber = *issue.Number
			issueExists = true
			break
		}
	}

	if issueExists {
		log.Printf("There is already an issue with the title '%s' in repo 'pulumi/%s'.", issueTitle, pulumiRepo)
	} else {
		log.Print("Did not find an existing issue.  Creating a new issue.")

		body := fmt.Sprintf("Release details: %s", event.Link)

		issue, _, err := gitHubClient.Issues.Create(ctx, "pulumi", pulumiRepo, &github.IssueRequest{
			Title:  github.String(issueTitle),
			Labels: &[]string{"kind/enhancement"},
			Body:   github.String(body),
		})
		if err != nil {
			return err
		}

		issueNumber = *issue.Number

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

	triggerWorkflowAllowList := strings.Split(os.Getenv("TRIGGER_WORKFLOW_ALLOW_LIST"), " ")
	if shouldTriggerWorkflow(pulumiRepo, triggerWorkflowAllowList) {
		log.Printf("Pulumi repo '%s' was found in the allow list. Triggering workflow.", pulumiRepo)
		repoObj, resp, err := gitHubClient.Repositories.Get(context.Background(), "pulumi", pulumiRepo)
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("Pulumi repo %s could not be found: %s", pulumiRepo, resp.Status)
		}
		if err != nil {
			return err
		}
		ref := repoObj.GetDefaultBranch()
		workflowParams := github.CreateWorkflowDispatchEventRequest{
			Ref: ref,
			Inputs: map[string]interface{}{
				"version":             version[1:], // strip the leading "v"
				"linked_issue_number": strconv.Itoa(issueNumber),
			},
		}

		log.Printf("Triggering workflow dispatch with parameters: %+v", workflowParams)
		_, err = gitHubClient.Actions.CreateWorkflowDispatchEventByFileName(ctx, "pulumi", pulumiRepo, "update-upstream-provider.yml", workflowParams)
		if err != nil {
			return err
		}
	} else {
		log.Printf("Pulumi repo '%s' is not contained in the allow list to automatically trigger the update workflow.", pulumiRepo)
	}

	log.Print("Done.")

	return nil
}

func isPreRelease(version string) bool {
	terms := []string{
		"pre",
		"beta",
		"alpha",
		"rc",
	}

	for _, term := range terms {
		if strings.Contains(version, term) {
			return true
		}
	}

	return false
}

func parseVersion(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", err
	}

	segments := strings.Split(u.Path, "/")
	return segments[len(segments)-1], nil
}

func parseTerraformRepo(terraformProviderUri string) (string, error) {
	u, err := url.Parse(terraformProviderUri)
	if err != nil {
		return "", err
	}

	segments := strings.Split(u.Path, "/")
	return segments[2], nil
}

func parsePulumiRepo(terraformProviderUri string) (string, error) {
	tfProvider, err := parseTerraformRepo(terraformProviderUri)
	if err != nil {
		return "", err
	}

	switch tfProvider {
	case "terraform-provider-azurerm":
		return "pulumi-azure", nil
	case "terraform-provider-confluent":
		return "pulumi-confluentcloud", nil
	case "terraform-provider-google-beta":
		return "pulumi-gcp", nil
	case "terraform-provider-bigip":
		return "pulumi-f5bigip", nil
	case "terraform":
		return "pulumi-terraform", nil
	default:
		return strings.Replace(tfProvider, "terraform-provider", "pulumi", -1), nil
	}
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

func shouldTriggerWorkflow(pulumiRepository string, allowList []string) bool {
	for _, item := range allowList {
		if strings.ToLower(strings.TrimSpace(item)) == strings.ToLower(strings.TrimSpace(pulumiRepository)) {
			return true
		}
	}

	return false
}
