package observe

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccObservePoller(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					icon_url  = "test"
				}
				resource "observe_poller" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					interval  = "1m"
					retries   = 5
					datastream = observe_datastream.example.oid
					skip_external_validation = true

					chunk {
					    enabled = true
						size = 1024
					}
					tags = {
						"k1"   = "v1"
						"k2"   = "v2"
					}
					http {
						method   = "POST"
						body   = jsonencode({ "hello" = "world" })
					    endpoint = "https://test.com"
						content_type = "application/json"
						headers = {
						    "token" = "test-token"
						}
					}
				}`, randomPrefix, "pollers", randomPrefix, "http"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_poller.first", "name", randomPrefix+"-http"),
					resource.TestCheckResourceAttr("observe_poller.first", "kind", "HTTP"),
					resource.TestCheckResourceAttr("observe_poller.first", "interval", "1m0s"),
					resource.TestCheckResourceAttr("observe_poller.first", "retries", "5"),
					resource.TestCheckResourceAttr("observe_poller.first", "tags.k1", "v1"),
					resource.TestCheckResourceAttr("observe_poller.first", "tags.k2", "v2"),
					resource.TestCheckResourceAttr("observe_poller.first", "chunk.0.enabled", "true"),
					resource.TestCheckResourceAttr("observe_poller.first", "chunk.0.size", "1024"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.method", "POST"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.body", `{"hello":"world"}`),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.endpoint", "https://test.com"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.content_type", "application/json"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.headers.token", "test-token"),
					resource.TestCheckResourceAttr("observe_poller.first", "pubsub.#", "0"),
					resource.TestCheckResourceAttrSet("observe_poller.first", "datastream"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					icon_url  = "test"
				}
				resource "observe_poller" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					interval  = "1m"
					retries   = 5
					datastream = observe_datastream.example.oid
					skip_external_validation = true

					http {
						template {
							username = "user"
							password = "pass"
							headers = {
								accept = "application/json"
							}
						}
						request {
							url = "https://example.com/path"
						}

						rule {
							match {
								url = "https://example.com/path"
							}

							decoder {
								type = "prometheus"
							}
						}
					}
				}`, randomPrefix, "pollers", randomPrefix, "http"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_poller.first", "name", randomPrefix+"-http"),
					resource.TestCheckResourceAttr("observe_poller.first", "kind", "HTTP"),
					resource.TestCheckResourceAttr("observe_poller.first", "interval", "1m0s"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.template.0.username", "user"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.template.0.password", "pass"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.url", "https://example.com/path"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.rule.0.match.0.url", "https://example.com/path"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.rule.0.decoder.0.type", "prometheus"),
					resource.TestCheckResourceAttr("observe_poller.first", "pubsub.#", "0"),
					resource.TestCheckResourceAttrSet("observe_poller.first", "datastream"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
                resource "observe_datastream" "example" {
                	workspace = data.observe_workspace.default.oid
                    name      = "%s-%s"
					icon_url  = "test"
                }
                resource "observe_poller" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
				    retries   = 5
				    datastream = observe_datastream.example.oid
					skip_external_validation = true

				    chunk {
					    enabled = true
						size = 1024
				    }
				    tags = {
						"k1"   = "v1"
						"k2"   = "v2"
				    }
				    pubsub {
					    project_id = "gcp-test"
					    subscription_id = "sub-test"
						json_key = jsonencode({
							type: "service_account",
							project_id: "gcp-test"
					    })
				    }
                }`, randomPrefix, "pollers", randomPrefix, "pubsub"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_poller.first", "name", randomPrefix+"-pubsub"),
					resource.TestCheckResourceAttr("observe_poller.first", "retries", "5"),
					resource.TestCheckResourceAttr("observe_poller.first", "tags.k1", "v1"),
					resource.TestCheckResourceAttr("observe_poller.first", "tags.k2", "v2"),
					resource.TestCheckResourceAttr("observe_poller.first", "chunk.0.enabled", "true"),
					resource.TestCheckResourceAttr("observe_poller.first", "chunk.0.size", "1024"),
					resource.TestCheckResourceAttr("observe_poller.first", "pubsub.0.project_id", "gcp-test"),
					resource.TestCheckResourceAttr("observe_poller.first", "pubsub.0.subscription_id", "sub-test"),
					resource.TestCheckResourceAttrSet("observe_poller.first", "pubsub.0.json_key"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.#", "0"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					icon_url  = "test"
				}
				resource "observe_poller" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					retries   = 5
					datastream = observe_datastream.example.oid
					skip_external_validation = true
					
					tags = {
						"k1"   = "v1"
						"k2"   = "v2"
					}
					gcp_monitoring {
					project_id = "gcp-test"
						json_key = jsonencode({
							type: "service_account",
							project_id: "gcp-test"
						})
						rate_limit = 50
						total_limit = 1000
					}
				}`, randomPrefix, "pollers", randomPrefix, "gcp"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_poller.first", "name", randomPrefix+"-gcp"),
					resource.TestCheckResourceAttr("observe_poller.first", "kind", "GCPMonitoring"),
					resource.TestCheckResourceAttr("observe_poller.first", "retries", "5"),
					resource.TestCheckResourceAttr("observe_poller.first", "tags.k1", "v1"),
					resource.TestCheckResourceAttr("observe_poller.first", "tags.k2", "v2"),
					resource.TestCheckResourceAttr("observe_poller.first", "gcp_monitoring.0.project_id", "gcp-test"),
					resource.TestCheckResourceAttrSet("observe_poller.first", "gcp_monitoring.0.json_key"),
					resource.TestCheckResourceAttr("observe_poller.first", "gcp_monitoring.0.rate_limit", "50"),
					resource.TestCheckResourceAttr("observe_poller.first", "gcp_monitoring.0.total_limit", "1000"),
					resource.TestCheckResourceAttr("observe_poller.first", "pubsub.#", "0"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.#", "0"),
				),
			},
		},
	})
}

func TestAccObservePollerMongoDB(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					icon_url  = "test"
				}
				resource "observe_poller" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s"
					interval  = "1m"
					datastream = observe_datastream.example.oid
					skip_external_validation = true

					tags = {
						"k1"   = "v1"
						"k2"   = "v2"
					}
					mongodbatlas {
						public_key  = "test"
						private_key = "test"
						exclude_groups = [
							"https://cloud.mongodb.com/users"
						]
					}
				}`, randomPrefix, "pollers", randomPrefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_poller.first", "name", randomPrefix),
					resource.TestCheckResourceAttr("observe_poller.first", "kind", "MongoDBAtlas"),
					resource.TestCheckResourceAttr("observe_poller.first", "interval", "1m0s"),
					resource.TestCheckResourceAttr("observe_poller.first", "tags.k1", "v1"),
					resource.TestCheckResourceAttr("observe_poller.first", "tags.k2", "v2"),
					resource.TestCheckResourceAttr("observe_poller.first", "mongodbatlas.0.public_key", "test"),
					resource.TestCheckResourceAttr("observe_poller.first", "mongodbatlas.0.private_key", "test"),
					resource.TestCheckResourceAttr("observe_poller.first", "mongodbatlas.0.exclude_groups.0", "https://cloud.mongodb.com/users"),
					resource.TestCheckResourceAttr("observe_poller.first", "gcp_monitoring.#", "0"),
					resource.TestCheckResourceAttr("observe_poller.first", "pubsub.#", "0"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.#", "0"),
				),
			},
		},
	})
}

func TestAccObservePollerHTTP(t *testing.T) {
	randomPrefix := acctest.RandomWithPrefix("tf")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					icon_url  = "test"
				}
				resource "observe_poller" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					disabled  = true
					interval  = "1m"
					retries   = 5
					datastream = observe_datastream.example.oid
					skip_external_validation = true

					http {
						request {
							url    = "https://example.com/path"
							method = "POST"
							username = "user"
							password = "pass"
							body = jsonencode({ "hello" = "world" })
						}
					}
				}`, randomPrefix, "pollers", randomPrefix, "http"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_poller.first", "name", randomPrefix+"-http"),
					resource.TestCheckResourceAttr("observe_poller.first", "disabled", "true"),
					resource.TestCheckResourceAttr("observe_poller.first", "interval", "1m0s"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.username", "user"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.password", "pass"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.url", "https://example.com/path"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.method", "POST"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.body", "{\"hello\":\"world\"}"),
					resource.TestCheckResourceAttr("observe_poller.first", "pubsub.#", "0"),
					resource.TestCheckResourceAttrSet("observe_poller.first", "datastream"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					icon_url  = "test"
				}
				resource "observe_poller" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					interval  = "1m"
					retries   = 5
					datastream = observe_datastream.example.oid
					skip_external_validation = true

					http {
						template {
							username = "user"
							password = "pass"
							headers = {
								"accept" = "application/json"
							}
						}
						request {
							url    = "https://example.com/path"
						}
					}
				}`, randomPrefix, "pollers", randomPrefix, "http"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_poller.first", "name", randomPrefix+"-http"),
					resource.TestCheckResourceAttr("observe_poller.first", "disabled", "false"),
					resource.TestCheckResourceAttr("observe_poller.first", "kind", "HTTP"),
					resource.TestCheckResourceAttr("observe_poller.first", "interval", "1m0s"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.template.0.username", "user"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.template.0.password", "pass"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.template.0.headers.accept", "application/json"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.template.0.params.#", "0"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.url", "https://example.com/path"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.method", ""),
					resource.TestCheckResourceAttr("observe_poller.first", "pubsub.#", "0"),
					resource.TestCheckResourceAttrSet("observe_poller.first", "datastream"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					icon_url  = "test"
				}
				resource "observe_poller" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					interval  = "1m"
					retries   = 5
					datastream = observe_datastream.example.oid
					skip_external_validation = true

					http {
						template {
							username = "user"
							password = "pass"
							auth_scheme = "digest"
							headers = {
								"accept" = "application/json"
							}
						}
						request {
							url    = "https://example.com/path"
						}
						
						rule {
							match {
								url    = "https://example.com/path"
							}
							follow = "accounts[]"
						}


					}
				}`, randomPrefix, "pollers", randomPrefix, "http"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_poller.first", "name", randomPrefix+"-http"),
					resource.TestCheckResourceAttr("observe_poller.first", "kind", "HTTP"),
					resource.TestCheckResourceAttr("observe_poller.first", "interval", "1m0s"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.template.0.username", "user"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.template.0.password", "pass"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.template.0.auth_scheme", "Digest"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.template.0.headers.accept", "application/json"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.template.0.params.#", "0"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.url", "https://example.com/path"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.method", ""),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.rule.0.match.0.url", "https://example.com/path"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.rule.0.match.0.method", ""),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.rule.0.follow", "accounts[]"),
					resource.TestCheckResourceAttr("observe_poller.first", "pubsub.#", "0"),
					resource.TestCheckResourceAttrSet("observe_poller.first", "datastream"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					icon_url  = "test"
				}
				resource "observe_poller" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					interval  = "1m"
					retries   = 5
					datastream = observe_datastream.example.oid
					skip_external_validation = true

					http {
						request {
							username    = "user"
							password    = "pass"
							auth_scheme = "Digest"
							url    = "https://example.com/path"
						}

						request {
							url    = "https://example.com/path2"
						}

						rule {
							match {
								url = "https://example.com/path"
							}

							decoder {
								type = "prometheus"
							}
						}

						rule {
							match {
								url = "https://example.com/path2"
							}
							follow = "accounts[]"
						}
					}
				}`, randomPrefix, "pollers", randomPrefix, "http"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_poller.first", "name", randomPrefix+"-http"),
					resource.TestCheckResourceAttr("observe_poller.first", "kind", "HTTP"),
					resource.TestCheckResourceAttr("observe_poller.first", "interval", "1m0s"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.template.#", "0"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.url", "https://example.com/path"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.auth_scheme", "Digest"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.1.url", "https://example.com/path2"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.rule.0.match.0.url", "https://example.com/path"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.rule.0.decoder.0.type", "prometheus"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.rule.0.follow", ""),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.rule.1.match.0.url", "https://example.com/path2"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.rule.1.follow", "accounts[]"),
					resource.TestCheckResourceAttr("observe_poller.first", "pubsub.#", "0"),
					resource.TestCheckResourceAttrSet("observe_poller.first", "datastream"),
				),
			},
			{
				Config: fmt.Sprintf(configPreamble+`
				resource "observe_datastream" "example" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					icon_url  = "test"
				}
				resource "observe_poller" "first" {
					workspace = data.observe_workspace.default.oid
					name      = "%s-%s"
					interval  = "1m"
					retries   = 5
					datastream = observe_datastream.example.oid
					skip_external_validation = true

					http {
						request {
							username    = "user"
							password    = "pass"
							auth_scheme = "Digest"
							url    = "https://example.com/path"
						}

						rule 
							match {
								url = "https://example.com/path2"
							}
							follow = "accounts[]"
						}

						timestamp {
							name = "now"
							format = "RFC822"
							truncate = "1s"
						}

						timestamp {
							name = "start"
							source = "now"
							format = "RFC822"
							offset = "1h"
							truncate = "1s"
						}
					}
				}`, randomPrefix, "pollers", randomPrefix, "http"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("observe_poller.first", "name", randomPrefix+"-http"),
					resource.TestCheckResourceAttr("observe_poller.first", "kind", "HTTP"),
					resource.TestCheckResourceAttr("observe_poller.first", "interval", "1m0s"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.template.#", "0"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.url", "https://example.com/path"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.0.auth_scheme", "Digest"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.request.1.url", "https://example.com/path2"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.timestamp.0.name", "a"),
					resource.TestCheckResourceAttr("observe_poller.first", "http.0.timestamp.1.name", "b"),
				),
			},
		},
	})
}
