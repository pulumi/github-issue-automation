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
}

func TestParseVersion(t *testing.T) {
	res, err := parseVersion("https://github.com/jfrog/terraform-provider-artifactory/releases/tag/v2.6.25")
	assert.Nil(t, err)
	assert.Equal(t, "v2.6.25", res)
}

func TestIsPreRelease(t *testing.T) {
	res1, err := isPreRelease("v0.25.29-pre%2B8224954")
	assert.Equal(t, true, res1)
	assert.Nil(t, err)

	res2, err := isPreRelease("v2.6.25")
	assert.Equal(t, false, res2)
	assert.Nil(t, err)
}
