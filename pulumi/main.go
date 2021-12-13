package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ssm"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		githubToken := os.Getenv("GITHUB_TOKEN")
		if githubToken == "" {
			panic("Environment variable GITHUB_TOKEN must be set to a GitHub token that allows the creation of issues in all repos in the Pulumi org.")
		}

		lambdaName := "new-release-handler"

		_, err := ssm.NewParameter(ctx, "release-handler-github-token", &ssm.ParameterArgs{
			Description: pulumi.String("GitHub token that allows the creation of issues in all repos in the Pulumi org."),
			Name:        pulumi.String(fmt.Sprintf("/%s/github-token", lambdaName)),
			Type:        ssm.ParameterTypeSecureString,
			Value:       pulumi.String(githubToken),
		}, pulumi.AdditionalSecretOutputs([]string{"value"}))
		if err != nil {
			return err
		}

		lambdaRole, err := iam.NewRole(ctx, "release-handler-role", &iam.RoleArgs{
			Name: pulumi.String("NewReleaseHandlerExecutionRole"),
			AssumeRolePolicy: pulumi.String(`{
				"Version": "2012-10-17",
				"Statement": [{
					"Sid": "",
					"Effect": "Allow",
					"Principal": {
						"Service": "lambda.amazonaws.com"
					},
					"Action": "sts:AssumeRole"
				}]
			}`),
		})
		if err != nil {
			return err
		}

		_, err = iam.NewRolePolicyAttachment(ctx, "release-handler-role-attachment", &iam.RolePolicyAttachmentArgs{
			PolicyArn: iam.ManagedPolicyAWSLambdaBasicExecutionRole,
			Role:      lambdaRole.Name,
		})
		if err != nil {
			return err
		}

		lambda, err := lambda.NewFunction(ctx, "new-release-handler", &lambda.FunctionArgs{
			Handler: pulumi.String(lambdaName),
			Code:    pulumi.NewFileArchive("../.build"),
			Role:    lambdaRole.Arn,
			Runtime: pulumi.String(lambda.RuntimeGo1dx),
			// Be careful not to change this value as it will break all existing Zaps
			// since Zapier invokes the Lambda by name:
			Name: pulumi.String(lambdaName),
		})
		if err != nil {
			return err
		}

		user, err := iam.NewUser(ctx, "zapier-user", &iam.UserArgs{
			Name: pulumi.String("zapier"),
		})
		if err != nil {
			return err
		}

		region, err := aws.GetRegion(ctx, nil, nil)
		if err != nil {
			return err
		}

		identity, err := aws.GetCallerIdentity(ctx, nil, nil)
		if err != nil {
			return err
		}

		ssmArn := fmt.Sprintf("arn:aws:ssm:%s:%s:parameter/%s/*", region.Name, identity.AccountId, lambdaName)
		bytes, err := json.Marshal((map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Effect": "Allow",
					"Action": []string{
						"ssm:GetParameterHistory",
						"ssm:GetParametersByPath",
						"ssm:GetParameters",
						"ssm:GetParameter",
					},
					"Resource": []string{
						ssmArn,
					},
				},
			},
		}))
		if err != nil {
			return err
		}

		policy, err := iam.NewPolicy(ctx, "read-ssm-params", &iam.PolicyArgs{
			Name:        pulumi.String("new-release-handler-read-ssm"),
			Description: pulumi.String("Allows the new-release-handler Lambda to access its SSM parameters"),
			Policy:      pulumi.String(bytes),
		})
		if err != nil {
			return err
		}

		_, err = iam.NewRolePolicyAttachment(ctx, "read-ssm-params-attachment", &iam.RolePolicyAttachmentArgs{
			PolicyArn: policy.Arn,
			Role:      lambdaRole.Name,
		})
		if err != nil {
			return err
		}

		policyDoc := lambda.Arn.ApplyT(func(arn string) (string, error) {
			bytes, jsonErr := json.Marshal((map[string]interface{}{
				"Version": "2012-10-17",
				"Statement": []map[string]interface{}{
					{
						"Effect": "Allow",
						"Action": []string{
							"lambda:InvokeFunction",
							"lambda:GetFunction",
						},
						"Resource": []string{
							arn,
						},
					},
					{
						"Effect": "Allow",
						"Action": []string{
							"lambda:ListFunctions",
						},
						"Resource": []string{
							"*",
						},
					},
				},
			}))
			if jsonErr != nil {
				return "", jsonErr
			}

			return string(bytes), nil
		})

		policy, err = iam.NewPolicy(ctx, "zapier-policy", &iam.PolicyArgs{
			Name:        pulumi.String("zapier"),
			Description: pulumi.String("Allows Zapier to invoke a Lambda"),
			Policy:      policyDoc,
		})
		if err != nil {
			return err
		}

		_, err = iam.NewUserPolicyAttachment(ctx, "zapier-policy-attachment", &iam.UserPolicyAttachmentArgs{
			PolicyArn: policy.Arn,
			User:      user.Name,
		})
		if err != nil {
			return err
		}

		return nil
	})
}
