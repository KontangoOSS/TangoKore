vault {
  address = "http://bao.tango:8200"
}

auto_auth {
  method "approle" {
    config = {
      role_id_file_path                   = "/opt/kontango/role-id"
      secret_id_file_path                 = "/opt/kontango/secret-id"
      remove_secret_id_file_after_reading = true
    }
  }

  sink "file" {
    config = {
      path = "/opt/kontango/bao-token"
      mode = 0600
    }
  }
}

template_config {
  static_secret_render_interval = "5m"
}

template {
  contents    = <<EOF
GREETING={{ with secret "secret/data/apps/hello-world" }}{{ .Data.data.greeting }}{{ end }}
SECRET_COLOR={{ with secret "secret/data/apps/hello-world" }}{{ .Data.data.color }}{{ end }}
SECRET_API_KEY={{ with secret "secret/data/apps/hello-world" }}{{ .Data.data.api_key }}{{ end }}
EOF
  destination = "/opt/kontango/profiles/hello-world/.env"
  perms       = 0600
  command     = "docker compose -f /opt/kontango/profiles/hello-world/compose.yml up -d --remove-orphans"
}
