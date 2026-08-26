// Copyright (c) 2026 Russell Obets
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/darkhonor/terraform-provider-technitium/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAppRecordParams(t *testing.T) {
	model := AppRecordResourceModel{
		AppName:    types.StringValue("Split Horizon"),
		ClassPath:  types.StringValue("SplitHorizon.SimpleAddress"),
		RecordData: types.StringValue(`{"10.0.0.0/8":["10.1.2.3"],"public":["192.0.2.3"]}`),
	}

	got := appRecordParams(&model)
	want := map[string]string{
		"appName":    "Split Horizon",
		"classPath":  "SplitHorizon.SimpleAddress",
		"recordData": `{"10.0.0.0/8":["10.1.2.3"],"public":["192.0.2.3"]}`,
	}
	for key, wantValue := range want {
		if got[key] != wantValue {
			t.Errorf("%s = %q, want %q", key, got[key], wantValue)
		}
	}
}

func TestAppRecordIdentityExcludesOpaqueData(t *testing.T) {
	if got, want := buildAppRecordID("example.com", "app.example.com"), "example.com::app.example.com"; got != want {
		t.Fatalf("buildAppRecordID() = %q, want %q", got, want)
	}
}

func TestAppRecordRData(t *testing.T) {
	record := client.Record{RData: map[string]interface{}{
		"appName":   "Split Horizon",
		"classPath": "SplitHorizon.SimpleAddress",
		"data":      `{"private":["10.1.2.3"]}`,
	}}

	if got := recordRDataString(record, "data"); got != `{"private":["10.1.2.3"]}` {
		t.Fatalf("record data = %q", got)
	}
}

func TestAccAppRecordResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAppRecord(`{"private":["10.1.2.3"]}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("technitium_app_record.test", "zone", "app-record-test.example.com"),
					resource.TestCheckResourceAttr("technitium_app_record.test", "name", "service.app-record-test.example.com"),
					resource.TestCheckResourceAttr("technitium_app_record.test", "app_name", "Split Horizon"),
					resource.TestCheckResourceAttr("technitium_app_record.test", "class_path", "SplitHorizon.SimpleAddress"),
					resource.TestCheckResourceAttr("technitium_app_record.test", "record_data", `{"private":["10.1.2.3"]}`),
					resource.TestCheckResourceAttr("technitium_app_record.test", "ttl", "300"),
					resource.TestMatchResourceAttr("technitium_app_record.test", "id",
						regexp.MustCompile(`^app-record-test\.example\.com::service\.app-record-test\.example\.com$`)),
				),
			},
			{
				Config: testAccAppRecord(`{"private":["10.1.2.4"]}`),
				Check: resource.TestCheckResourceAttr(
					"technitium_app_record.test", "record_data", `{"private":["10.1.2.4"]}`),
			},
			{
				ResourceName:      "technitium_app_record.test",
				ImportState:       true,
				ImportStateId:     "app-record-test.example.com::service.app-record-test.example.com",
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAppRecord(recordData string) string {
	return testAccProviderHCL() + fmt.Sprintf(`

resource "technitium_zone" "test" {
  name = "app-record-test.example.com"
  type = "Primary"

  dnssec {
    enabled = false
  }
}

resource "technitium_app_record" "test" {
  zone        = technitium_zone.test.name
  name        = "service.app-record-test.example.com"
  ttl         = 300
  app_name    = "Split Horizon"
  class_path  = "SplitHorizon.SimpleAddress"
  record_data = %q
}
`, recordData)
}
