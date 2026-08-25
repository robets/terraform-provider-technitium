---
subcategory: ""
page_title: "technitium_app_record Resource - terraform-provider-technitium"
description: |-
  Manages a Technitium APP record. The referenced DNS application must already be installed
  on the server.
---

# technitium\_app\_record (Resource)

Manages the singular Technitium APP record at a DNS name. The referenced DNS application
must already be installed on the server.

APP records are exclusive with ordinary DNS records at the same name and are not supported
in DNSSEC-signed primary zones. `record_data` is application-defined and may be empty or use
a format other than JSON; use `jsonencode` when the selected handler expects JSON.

## Example Usage

```terraform
resource "technitium_app_record" "service" {
  zone       = "example.com"
  name       = "service.example.com"
  ttl        = 300
  app_name   = "Split Horizon"
  class_path = "SplitHorizon.SimpleAddress"

  record_data = jsonencode({
    "10.0.0.0/8"    = ["10.1.2.10"]
    "100.64.0.0/10" = ["100.64.1.10"]
    "public"        = ["192.0.2.10"]
  })
}
```

## Import

Import an APP record with its zone and fully qualified name separated by `::`:

```shell
terraform import technitium_app_record.service 'example.com::service.example.com'
```
