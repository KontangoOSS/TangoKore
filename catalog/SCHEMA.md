# Catalog Schema

Standard formats for all catalog entries. Each file is a single YAML document with flat key-value pairs — no nesting.

File names are the **slug** (lowercase, hyphens, DNS-safe). The slug is the canonical identifier for that entry across the entire platform.

---

## Applications (`catalog/applications/{slug}.yaml`)

### Required Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `name` | string | Display name | `Ticketarr` |
| `description` | string | One-line summary | `Project management and ticketing` |
| `category` | ref | Slug from `catalog/categories/` | `operations` |
| `license` | ref | Slug from `catalog/licenses/` | `mit` |
| `language` | ref | Primary language, slug from `catalog/tech/` | `go` |
| `maintainer` | ref | Slug from `catalog/maintainers/` | `kontango` |
| `source` | string | Org/repo path (`kore/` for internal, `external/` for third-party) | `kore/ticketarr` |
| `official_site` | url | Project homepage | `https://github.com/KontangoOSS/ticketarr` |
| `default_ports` | string | Comma-separated port list | `9090,22` |
| `default_build` | string | Default image tag or version | `latest` |
| `virtualization_type` | enum | `lxc`, `vm`, or `bare` | `lxc` |
| `sizing_min` | ref | Minimum sizing profile from `catalog/sizing/` | `tg-sm-1` |
| `sizing_suggested` | ref | Recommended sizing profile | `tg-sm-2` |
| `sizing_max` | ref | Maximum sizing profile | `tg-md-1` |
| `active` | bool | Whether this app is available for deployment | `true` |

### Optional Fields

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `docker_image` | string | Full Docker image reference | `grafana/grafana:latest` |
| `frontend` | ref | Frontend tech, slug from `catalog/tech/` | `vue` |
| `framework` | ref | Framework, slug from `catalog/tech/` | `gin` |
| `depends_on` | string | Comma-separated app slugs this app requires | `postgres-db,redis` |
| `repo_url` | url | Source repository URL (if different from official_site) | `https://github.com/org/repo` |

### Ref Fields

Fields marked as `ref` are slugs that reference another catalog file. For example, `category: operations` refers to `catalog/categories/operations.yaml`. This keeps the data flat and portable — no Bao paths or internal URLs.

### Example

```yaml
# catalog/applications/ticketarr.yaml
active: true
category: operations
default_build: latest
default_ports: 9090,22
description: Project management and ticketing
frontend: vue
language: go
license: mit
maintainer: kontango
name: Ticketarr
official_site: "https://github.com/KontangoOSS/ticketarr"
sizing_max: tg-md-1
sizing_min: tg-sm-1
sizing_suggested: tg-sm-2
source: kore/ticketarr
virtualization_type: lxc
```

---

## Categories (`catalog/categories/{slug}.yaml`)

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name |
| `description` | string | One-line summary |
| `vmid_prod_start` | int | VMID range start (prod) |
| `vmid_prod_end` | int | VMID range end (prod) |
| `vmid_dev_start` | int | VMID range start (dev) |
| `vmid_dev_end` | int | VMID range end (dev) |

---

## OS (`catalog/os/{family}/{distro}.yaml`)

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name |
| `family` | enum | `linux`, `darwin`, `windows` |
| `distro` | string | Distribution identifier |
| `maintainer` | string | Who maintains this distro |
| `type` | enum | `server`, `desktop`, `container`, `hypervisor`, `kubernetes`, `container-os` |

Optional: `based_on`, `immutable`

---

## Sizing (`catalog/sizing/{slug}.yaml`)

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Display name |
| `cpu` | int | vCPU count |
| `memory_gb` | int | RAM in GB |
| `storage_gb` | int | Disk in GB |
| `description` | string | When to use this tier |

---

## Other Sections

**Licenses**, **Maintainers**, **Providers**, **Tech**, **Environments** follow the same flat YAML pattern with `name` + `description` as minimum fields. Check existing files for the current format.

---

## Contributing a New Application

1. Create `catalog/applications/{slug}.yaml` with all required fields
2. Ensure referenced categories, licenses, tech, and maintainers exist (or add them)
3. Submit a pull request
4. CI validates the schema and syncs to the platform catalog
