package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseTerraformRepo(t *testing.T) {
	res, err := parseTerraformRepo("https://github.com/jfrog/terraform-provider-artifactory/releases/tag/v2.6.21")
	assert.Nil(t, err)
	assert.Equal(t, "terraform-provider-artifactory", res)

	res, err = parseTerraformRepo("https://github.com/hashicorp/terraform-provider-azurerm/releases/tag/v2.6.21")
	assert.Nil(t, err)
	assert.Equal(t, "terraform-provider-azurerm", res)

	res, err = parseTerraformRepo("https://github.com/hashicorp/terraform-provider-google-beta/releases/tag/v2.6.21")
	assert.Nil(t, err)
	assert.Equal(t, "terraform-provider-google-beta", res)
}

func TestParsePulumiRepo(t *testing.T) {
	res, err := parsePulumiRepo("https://github.com/jfrog/terraform-provider-artifactory/releases/tag/v2.6.21")
	assert.Nil(t, err)
	assert.Equal(t, "pulumi-artifactory", res)

	res, err = parsePulumiRepo("https://github.com/hashicorp/terraform-provider-azurerm/releases/tag/v2.6.21")
	assert.Nil(t, err)
	assert.Equal(t, "pulumi-azure", res)

	res, err = parsePulumiRepo("https://github.com/hashicorp/terraform-provider-google-beta/releases/tag/v2.6.21")
	assert.Nil(t, err)
	assert.Equal(t, "pulumi-gcp", res)

	res, err = parsePulumiRepo("https://github.com/hashicorp/terraform/releases/tag/v1.1.3")
	assert.Nil(t, err)
	assert.Equal(t, "pulumi-terraform", res)

	res, err = parsePulumiRepo("https://github.com/F5Networks/terraform-provider-bigip/releases/tag/v1.13.0")
	assert.Nil(t, err)
	assert.Equal(t, "pulumi-f5bigip", res)

	res, err = parsePulumiRepo("https://github.com/confluentinc/terraform-provider-confluent/releases/tag/v1.13.0")
	assert.Nil(t, err)
	assert.Equal(t, "pulumi-confluentcloud", res)
}

func TestParseVersion(t *testing.T) {
	res, err := parseVersion("https://github.com/jfrog/terraform-provider-artifactory/releases/tag/v2.6.25")
	assert.Nil(t, err)
	assert.Equal(t, "v2.6.25", res)
}

func TestIsPreRelease(t *testing.T) {
	assert.Equal(t, true, isPreRelease("v0.25.29-pre%2B8224954"))
	assert.Equal(t, false, isPreRelease("v2.6.25"))
	assert.Equal(t, true, isPreRelease("v2.41.0-beta.2"))
	assert.Equal(t, true, isPreRelease("v1.2.0-alpha-20220328"))
	assert.Equal(t, true, isPreRelease("v1.2.0-rc2"))
	assert.Equal(t, true, isPreRelease("v1.2.0-rc2-rc2"))
}

func TestShouldTriggerWorkflow(t *testing.T) {
	res1 := shouldTriggerWorkflow("foo", []string{"bar", "  FOO     "})
	assert.True(t, res1)

	res2 := shouldTriggerWorkflow("foo", []string{"bar"})
	assert.False(t, res2)
}
