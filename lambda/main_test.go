package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

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
