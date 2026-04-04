# Profiles

A profile is a deployable unit — everything a machine needs to run an application. The controller pushes profiles to machines. The machine pulls the bundle, configures the Bao agent, and starts the application.

## Structure

```
profiles/my-app/
  manifest.yaml     # what to deploy, where, which secrets
  bao-agent.hcl     # OpenBao agent config (template rendering)
  compose.yml       # Docker Compose definition
```

## manifest.yaml

```yaml
profile: my-app
version: v1.0.0
description: My application

files:
  - path: bao-agent.hcl
    dest: /opt/kontango/bao-agent.hcl
  - path: compose.yml
    dest: /opt/kontango/profiles/my-app/compose.yml

secrets:
  path: secret/data/apps/my-app
  keys:
    - database_url
    - api_key
    - redis_url

restart:
  - kontango-bao-agent
```

## bao-agent.hcl

The Bao agent config tells OpenBao which secrets to fetch and how to render them. Secrets are injected as environment variables — nothing written to disk.

```hcl
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

template {
  source      = "/opt/kontango/profiles/my-app/env.tpl"
  destination = "/opt/kontango/profiles/my-app/.env"
  perms       = 0600
  command     = "docker compose -f /opt/kontango/profiles/my-app/compose.yml up -d --force-recreate"
}
```

## compose.yml

Standard Docker Compose. Reads secrets from the `.env` file rendered by the Bao agent:

```yaml
services:
  app:
    image: my-app:latest
    env_file: .env
    restart: unless-stopped
    ports:
      - "127.0.0.1:8080:8080"
```

The application reads `os.Getenv("DATABASE_URL")` — no changes needed.

## Deploy Flow

```sh
# Push a profile to a specific machine
curl -X POST https://controller.tango/api/deploy \
  -d '{
    "profile": "my-app",
    "version": "v1.0.0",
    "target": "eager-phoenix",
    "bao_role": "my-app"
  }'

# Push to all machines with a specific role
curl -X POST https://controller.tango/api/deploy \
  -d '{
    "profile": "my-app",
    "version": "v1.0.0",
    "target": "#web-servers",
    "bao_role": "my-app"
  }'
```

## Secret Rotation

When a secret changes in OpenBao, the Bao agent detects the change (polling interval is configurable, default 5 minutes), re-renders the template, and runs the `command` — which recreates the Docker container with new environment variables. Zero downtime if the application handles SIGTERM gracefully.

## Example: hello-world

See [`profiles/hello-world/`](../profiles/hello-world/) for a complete working example that demonstrates secret injection into a busybox container.
