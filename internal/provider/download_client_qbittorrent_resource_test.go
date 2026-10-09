package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/devopsarr/terraform-provider-prowlarr/internal/helpers"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

func TestAccDownloadClientQbittorrentResource(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Unauthorized Create
			{
				Config:      testAccDownloadClientQbittorrentResourceConfig("resourceQbittorrentTest", "qbittorrent") + testUnauthorizedProvider,
				ExpectError: regexp.MustCompile("Client Error"),
			},
			// Create and Read testing
			{
				Config: testAccDownloadClientQbittorrentResourceConfig("resourceQbittorrentTest", "qbittorrent"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prowlarr_download_client_qbittorrent.test", "host", "qbittorrent"),
					resource.TestCheckResourceAttr("prowlarr_download_client_qbittorrent.test", "url_base", "/qbittorrent/"),
					resource.TestCheckResourceAttrSet("prowlarr_download_client_qbittorrent.test", "id"),
				),
			},
			// Unauthorized Read
			{
				Config:      testAccDownloadClientQbittorrentResourceConfig("resourceQbittorrentTest", "qbittorrent") + testUnauthorizedProvider,
				ExpectError: regexp.MustCompile("Client Error"),
			},
			// Update and Read testing
			{
				Config: testAccDownloadClientQbittorrentResourceConfig("resourceQbittorrentTest", "qbittorrent-host"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prowlarr_download_client_qbittorrent.test", "host", "qbittorrent-host"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "prowlarr_download_client_qbittorrent.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccDownloadClientQbittorrentResourceConfig(name, host string) string {
	return fmt.Sprintf(`
	resource "prowlarr_download_client_qbittorrent" "test" {
		enable = false
		priority = 1
		name = "%s"
		host = "%s"
		url_base = "/qbittorrent/"
		port = 9091
		category = "tv-prowlarr"
		categories = [
			{
				name = "test"
				categories = [1000]
			}
		]
	}`, name, host)
}

// Needs a qBittorrent 5.2 or later at QBITTORRENT_HOST:QBITTORRENT_PORT that
// the app can reach, with QBITTORRENT_API_KEY as its Web UI API key. The
// client is enabled, so the app tests the connection with the key on every
// save.
func TestAccDownloadClientQbittorrentResourceAPIKeyWO(t *testing.T) {
	t.Parallel()

	host, port, key := os.Getenv("QBITTORRENT_HOST"), os.Getenv("QBITTORRENT_PORT"), os.Getenv("QBITTORRENT_API_KEY")
	if host == "" || port == "" || key == "" {
		t.Skip("QBITTORRENT_HOST, QBITTORRENT_PORT and QBITTORRENT_API_KEY must be set for this test")
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		TerraformVersionChecks: []tfversion.TerraformVersionCheck{
			tfversion.SkipBelow(tfversion.Version1_11_0),
		},
		Steps: []resource.TestStep{
			// Create with the key, which the app tests against qBittorrent
			{
				Config: testAccDownloadClientQbittorrentResourceAPIKeyWOConfig("resourceQbittorrentKeyTest", host, port, key, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("prowlarr_download_client_qbittorrent.test", "api_key_wo"),
					resource.TestCheckResourceAttr("prowlarr_download_client_qbittorrent.test", "api_key_wo_version", "1"),
					resource.TestCheckResourceAttr("prowlarr_download_client_qbittorrent.test", "api_key", helpers.SensitiveValue),
				),
			},
			// Update something else: the key is not sent, and the app keeps it
			{
				Config: testAccDownloadClientQbittorrentResourceAPIKeyWOConfig("resourceQbittorrentKeyRenamed", host, port, key, 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prowlarr_download_client_qbittorrent.test", "name", "resourceQbittorrentKeyRenamed"),
				),
			},
			// A new version sends the new key: a wrong one fails the app's test
			{
				Config:      testAccDownloadClientQbittorrentResourceAPIKeyWOConfig("resourceQbittorrentKeyRenamed", host, port, "qbt_WrongKey23456789ABCDEFGHJKL", 2),
				ExpectError: regexp.MustCompile("(?i)Authentication failure"),
			},
			// And the right one passes it
			{
				Config: testAccDownloadClientQbittorrentResourceAPIKeyWOConfig("resourceQbittorrentKeyRenamed", host, port, key, 3),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prowlarr_download_client_qbittorrent.test", "api_key_wo_version", "3"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccDownloadClientQbittorrentResourceAPIKeyWOConfig(name, host, port, key string, version int) string {
	return fmt.Sprintf(`
	resource "prowlarr_download_client_qbittorrent" "test" {
		enable = true
		priority = 1
		name = "%s"
		host = "%s"
		port = %s
		username = ""
		password = ""
		api_key_wo = "%s"
		api_key_wo_version = %d
		category = "prowlarr"
	}`, name, host, port, key, version)
}
