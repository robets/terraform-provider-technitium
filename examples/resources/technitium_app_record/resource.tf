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
