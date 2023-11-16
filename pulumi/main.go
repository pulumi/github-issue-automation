package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/lambda"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/secretsmanager"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() { pulumi.Run(runPulumi) }

func runPulumi(ctx *pulumi.Context) error {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		panic("Environment variable GITHUB_TOKEN must be set to a GitHub token that allows the creation of issues in all repos in the Pulumi org.")
	}

	lambdaName := "new-release-handler"

	secret, err := secretsmanager.NewSecret(ctx, "release-handler-github-token-secret", &secretsmanager.SecretArgs{
		Description: pulumi.String("GitHub token that allows the creation of issues in all repos in the Pulumi org."),
		NamePrefix:  pulumi.String(fmt.Sprintf("/%s/github-token", lambdaName)),
	})
	if err != nil {
		return err
	}

	_, err = secretsmanager.NewSecretVersion(ctx, "release-handler-github-token-version", &secretsmanager.SecretVersionArgs{
		SecretId:     secret.ID(),
		SecretString: pulumi.String(githubToken),
	})
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

	lambdaFunction, err := lambda.NewFunction(ctx, "new-release-handler", &lambda.FunctionArgs{
		Handler: pulumi.String(lambdaName),
		Code:    pulumi.NewFileArchive("../.build/new-release-handler.zip"),
		Role:    lambdaRole.Arn,
		Runtime: pulumi.String(lambda.RuntimeGo1dx),
		// Be careful not to change this value as it will break all existing Zaps
		// since Zapier invokes the Lambda by name:
		Name: pulumi.String(lambdaName),
		Environment: &lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap{
				"GITHUB_TOKEN_SECRET_ARN": secret.Arn,
			},
		},
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

	// Source: https://docs.aws.amazon.com/mediaconnect/latest/ug/iam-policy-examples-asm-secrets.html#iam-policy-examples-asm-specific-secrets
	secretPolicyDoc := secret.Arn.ApplyT(func(arn string) (string, error) {
		bytes, err := json.Marshal(map[string]interface{}{
			"Version": "2012-10-17",
			"Statement": []map[string]interface{}{
				{
					"Effect": "Allow",
					"Action": []string{
						"secretsmanager:GetResourcePolicy",
						"secretsmanager:GetSecretValue",
						"secretsmanager:DescribeSecret",
						"secretsmanager:ListSecretVersionIds",
					},
					"Resource": []string{
						arn,
					},
				},
				{
					"Effect":   "Allow",
					"Action":   "secretsmanager:ListSecrets",
					"Resource": "*",
				},
			},
		})
		if err != nil {
			return "", err
		}

		return string(bytes), nil
	})
	if err != nil {
		return err
	}

	secretsPolicy, err := iam.NewPolicy(ctx, "new-release-handler-read-secrets", &iam.PolicyArgs{
		Description: pulumi.String("Allows the new-release-handler Lambda to access its secrets."),
		Policy:      secretPolicyDoc,
	})
	if err != nil {
		return err
	}

	_, err = iam.NewRolePolicyAttachment(ctx, "read-secrets-attachment", &iam.RolePolicyAttachmentArgs{
		PolicyArn: secretsPolicy.Arn,
		Role:      lambdaRole.Name,
	})
	if err != nil {
		return err
	}

	zapierPolicyDoc := lambdaFunction.Arn.ApplyT(func(arn string) (string, error) {
		bytes, jsonErr := json.Marshal(map[string]interface{}{
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
		})
		if jsonErr != nil {
			return "", jsonErr
		}

		return string(bytes), nil
	})

	zapierPolicy, err := iam.NewPolicy(ctx, "zapier-policy", &iam.PolicyArgs{
		Name:        pulumi.String("zapier"),
		Description: pulumi.String("Allows Zapier to invoke a Lambda"),
		Policy:      zapierPolicyDoc,
	})
	if err != nil {
		return err
	}

	_, err = iam.NewUserPolicyAttachment(ctx, "zapier-policy-attachment", &iam.UserPolicyAttachmentArgs{
		PolicyArn: zapierPolicy.Arn,
		User:      user.Name,
	})
	if err != nil {
		return err
	}

	err = createInternalReleaseHandler(ctx, lambdaRole, secret)
	if err != nil {
		return err
	}

	return nil
}

func createInternalReleaseHandler(ctx *pulumi.Context, lambdaRole *iam.Role, secret *secretsmanager.Secret) error {
	lambdaFunction, err := lambda.NewFunction(ctx, "internal-release-handler", &lambda.FunctionArgs{
		// The function is called by name in GHA, so we have to specify explicitly:
		Name:    pulumi.String("internal-release-handler"),
		Handler: pulumi.String("internal-release-handler"),
		Code:    pulumi.NewFileArchive("../.build/internal-release-handler.zip"),
		Role:    lambdaRole.Arn,
		Runtime: pulumi.String(lambda.RuntimeGo1dx),
		Environment: &lambda.FunctionEnvironmentArgs{
			Variables: pulumi.StringMap{
				"GITHUB_TOKEN_SECRET_ARN": secret.Arn,
			},
		},
	})
	if err != nil {
		return err
	}

	user, err := iam.NewUser(ctx, "internal-release-handler-user", &iam.UserArgs{
		Name: pulumi.String("internal-release-handler-user"),
	})
	if err != nil {
		return err
	}

	userPolicyDoc := lambdaFunction.Arn.ApplyT(func(arn string) (string, error) {
		bytes, jsonErr := json.Marshal(map[string]interface{}{
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
		})
		if jsonErr != nil {
			return "", jsonErr
		}

		return string(bytes), nil
	})

	policy, err := iam.NewPolicy(ctx, "internal-release-handler-policy", &iam.PolicyArgs{
		Description: pulumi.String("Invoke internal-release-handler lambda"),
		Policy:      userPolicyDoc,
	})

	_, err = iam.NewUserPolicyAttachment(ctx, "internal-release-user-policy-attachment", &iam.UserPolicyAttachmentArgs{
		PolicyArn: policy.Arn,
		User:      user.Name,
	})
	if err != nil {
		return err
	}

	return nil
}
