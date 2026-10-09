package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/devopsarr/terraform-provider-prowlarr/internal/helpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccApplicationReadarrResource(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Unauthorized Create
			{
				Config:      testAccApplicationReadarrResourceConfig("resourceReadarrTest", "false") + testUnauthorizedProvider,
				ExpectError: regexp.MustCompile("Client Error"),
			},
			// Create and Read testing
			{
				Config: testAccApplicationReadarrResourceConfig("resourceReadarrTest", "false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prowlarr_application_readarr.test", "prowlarr_url", "false"),
					resource.TestCheckResourceAttrSet("prowlarr_application_readarr.test", "id"),
				),
			},
			// Unauthorized Read
			{
				Config:      testAccApplicationReadarrResourceConfig("resourceReadarrTest", "false") + testUnauthorizedProvider,
				ExpectError: regexp.MustCompile("Client Error"),
			},
			// Update and Read testing
			{
				Config: testAccApplicationReadarrResourceConfig("resourceReadarrTest", "true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prowlarr_application_readarr.test", "prowlarr_url", "true"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "prowlarr_application_readarr.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"api_key"},
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccApplicationReadarrResourceConfig(name, prowlarr string) string {
	return fmt.Sprintf(`
	resource "prowlarr_application_readarr" "test" {
		name = "%s"
		sync_level = "disabled"

		base_url = "http://localhost:8787"
		prowlarr_url = "%s"
		api_key = "APIKey"
		sync_categories = [2010, 2020]
	}`, name, prowlarr)
}

func TestAccApplicationReadarrResourceAPIKeyWO(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		Steps: []resource.TestStep{
			// Create with the write-only key: only the masked value is stored
			{
				Config: testAccApplicationReadarrResourceAPIKeyWOConfig("resourceReadarrKeyTest", "APIKey", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("prowlarr_application_readarr.test", "api_key_wo"),
					resource.TestCheckResourceAttr("prowlarr_application_readarr.test", "api_key_wo_version", "1"),
					resource.TestCheckResourceAttr("prowlarr_application_readarr.test", "api_key", helpers.SensitiveValue),
				),
			},
			// Update something else: the key is not sent, and the app keeps it
			{
				Config: testAccApplicationReadarrResourceAPIKeyWOConfig("resourceReadarrKeyRenamed", "APIKey", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prowlarr_application_readarr.test", "name", "resourceReadarrKeyRenamed"),
				),
			},
			// A new version sends the new key
			{
				Config: testAccApplicationReadarrResourceAPIKeyWOConfig("resourceReadarrKeyRenamed", "OtherKey", 2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prowlarr_application_readarr.test", "api_key_wo_version", "2"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccApplicationReadarrResourceAPIKeyWOConfig(name, key string, version int) string {
	return fmt.Sprintf(`
	resource "prowlarr_application_readarr" "test" {
		name = "%s"
		sync_level = "disabled"

		base_url = "http://localhost:8787"
		prowlarr_url = "http://localhost:9696"
		api_key_wo = "%s"
		api_key_wo_version = %d
		sync_categories = [2010, 2020]
	}`, name, key, version)
}
