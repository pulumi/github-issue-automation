package main

import (
	"encoding/json"
	"log"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

type NewRelease struct {
	Title string `json:Title`
	Link  string `json:Link`
	// Tracks the name of the Zap that originated the request:
	ZapierSource string `json:ZapierSource`
}

func main() {
	lambda.Start(LambdaHandler)
}

func LambdaHandler(event NewRelease) error {
	bytes, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return err
	}

	log.Printf("Event = %s", bytes)

	session, err := session.NewSession()
	if err != nil {
		return err
	}

	ssmClient := ssm.New(session)

	_, err = ssmClient.GetParameter(&ssm.GetParameterInput{
		Name:           aws.String("/new-release-handler/github-token"),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return err
	}

	pulumiRepo, err := parsePulumiRepo(event.Link)
	if err != nil {
		return err
	}

	log.Printf("Creating an issue in repo pulumi/%s", pulumiRepo)

	return nil
}

func parsePulumiRepo(terraformProviderUri string) (string, error) {
	u, err := url.Parse(terraformProviderUri)
	if err != nil {
		return "", err
	}

	segments := strings.Split(u.Path, "/")
	tfProvider := segments[2]

	switch tfProvider {
	case "terraform-provider-azurerm":
		return "pulumi-azure", nil
	case "terraform-provider-google-beta":
		return "pulumi-gcp", nil
	default:
		return strings.Replace(tfProvider, "terraform-provider", "pulumi", -1), nil
	}
}
