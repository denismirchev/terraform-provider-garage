# Terraform Provider for Garage

A lightweight Terraform provider for Garage storage using the v1 admin API.

[![CI](https://github.com/d0ugal/terraform-provider-garage/actions/workflows/ci.yml/badge.svg)](https://github.com/d0ugal/terraform-provider-garage/actions/workflows/ci.yml)

## Features

### Resources

- **garage_key**: Create and manage access keys (with bucket creation permission support)
- **garage_bucket**: Create and manage buckets (with website configuration and quotas)
- **garage_bucket_key**: Manage key permissions on buckets
- **garage_bucket_alias**: Add and remove global or local bucket aliases

### Data Sources

- **garage_key**: Look up a single access key by ID or search
- **garage_keys**: List all access keys
- **garage_bucket**: Look up a single bucket by ID, alias, or search
- **garage_buckets**: List all buckets
- **garage_cluster_status**: Get cluster status with node information
- **garage_cluster_health**: Get cluster health metrics

## Building

```bash
make build
make install
```

## Usage

```hcl
terraform {
  required_providers {
    garage = {
      source  = "d0ugal/garage"
      version = "0.0.1"
    }
  }
}

provider "garage" {
  scheme = "http"
  host   = "127.0.0.1:3903"
  token  = "your-admin-token"
}

resource "garage_key" "loki_key" {
  name               = "loki-access-key"
  allow_create_bucket = true
}

resource "garage_bucket" "loki" {
  global_alias = "loki"

  # Optional quotas (0 = unlimited)
  max_size    = 10737418240  # 10 GiB in bytes
  max_objects = 100000

  # Configure website access
  website_access_enabled        = true
  website_access_index_document = "index.html"
  website_access_error_document = "error.html"
}

resource "garage_bucket_alias" "loki_local" {
  bucket_id     = garage_bucket.loki.id
  alias         = "loki-local"
  access_key_id = garage_key.loki_key.access_key_id
}

resource "garage_bucket_key" "loki_access" {
  bucket_id     = garage_bucket.loki.id
  access_key_id = garage_key.loki_key.access_key_id
  read          = true
  write         = true
  owner         = false
}

data "garage_cluster_status" "this" {}

data "garage_cluster_health" "this" {}

data "garage_keys" "all" {}
```

## Resources

### garage_key

Create and manage Garage access keys.

```hcl
resource "garage_key" "example" {
  name                = "my-access-key"
  allow_create_bucket = false  # optional, default false
}
```

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | yes | Name of the access key |
| `allow_create_bucket` | bool | no | Whether the key is allowed to create buckets (default `false`) |

**Attributes:**

| Name | Description |
|------|-------------|
| `id` | The key ID |
| `access_key_id` | The access key ID |
| `secret_access_key` | The secret access key |

### garage_bucket

Create and manage Garage buckets, including website configuration and quotas.

```hcl
resource "garage_bucket" "example" {
  global_alias = "my-bucket"

  website_config {
    enabled         = true
    index_document  = "index.html"
    error_document  = "error.html"
  }

  quotas {
    max_size    = 10737418240
    max_objects = 1000000
  }
}
```

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `global_alias` | string | no | Global alias for the bucket |
| `website_config` | block | no | Website configuration (see below) |
| `quotas` | block | no | Bucket quotas (see below) |

**`website_config` block:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `enabled` | bool | yes | Enable website hosting |
| `index_document` | string | yes | Index document for the website |
| `error_document` | string | no | Error document for the website |

**`quotas` block:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `max_size` | number | yes | Maximum bucket size in bytes |
| `max_objects` | number | yes | Maximum number of objects in the bucket |

**Attributes:**

| Name | Description |
|------|-------------|
| `id` | The bucket ID |
| `bytes` | Current bucket size in bytes |
| `objects` | Current number of objects in the bucket |
| `expiration_days` | Bucket expiration in days |

### garage_bucket_key

Manage key permissions on a bucket.

```hcl
resource "garage_bucket_key" "example" {
  bucket_id     = garage_bucket.example.id
  access_key_id = garage_key.example.access_key_id
  read          = true
  write         = true
  owner         = false
}
```

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `bucket_id` | string | yes | The bucket ID |
| `access_key_id` | string | yes | The access key ID |
| `read` | bool | yes | Allow read access |
| `write` | bool | yes | Allow write access |
| `owner` | bool | yes | Grant owner permissions |

### garage_bucket_alias

Add or remove global and local bucket aliases. Global aliases are accessible without authentication. Local aliases are scoped to a specific access key.

```hcl
# Global alias
resource "garage_bucket_alias" "example_global" {
  bucket_id = garage_bucket.example.id
  alias     = "my-bucket-alias"
}

# Local alias (scoped to an access key)
resource "garage_bucket_alias" "example_local" {
  bucket_id     = garage_bucket.example.id
  alias         = "my-bucket-local"
  access_key_id = garage_key.example.access_key_id
}
```

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `bucket_id` | string | yes | The bucket ID |
| `alias` | string | yes | The alias name |
| `access_key_id` | string | no | The access key ID (required for local aliases, omit for global) |

## Data Sources

### garage_key

Look up a single access key by ID or search by name.

```hcl
# By ID
data "garage_key" "by_id" {
  id = "GKabc123..."
}

# By search
data "garage_key" "by_name" {
  search = "my-key-name"
}
```

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `id` | string | no | The key ID (conflicts with `search`) |
| `search` | string | no | Search pattern to find the key (conflicts with `id`) |

**Attributes:**

| Name | Description |
|------|-------------|
| `id` | The key ID |
| `name` | The key name |
| `access_key_id` | The access key ID |
| `allow_create_bucket` | Whether the key can create buckets |

### garage_keys

List all access keys in the cluster.

```hcl
data "garage_keys" "all" {}
```

**Attributes:**

| Name | Description |
|------|-------------|
| `keys` | List of all access keys |

### garage_bucket

Look up a single bucket by ID, alias, or search.

```hcl
# By ID
data "garage_bucket" "by_id" {
  id = "bUCKETabc123..."
}

# By alias
data "garage_bucket" "by_alias" {
  alias = "my-bucket"
}

# By search
data "garage_bucket" "by_name" {
  search = "my-bucket"
}
```

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `id` | string | no | The bucket ID (conflicts with `alias` and `search`) |
| `alias` | string | no | The bucket alias (conflicts with `id` and `search`) |
| `search` | string | no | Search pattern (conflicts with `id` and `alias`) |

**Attributes:**

| Name | Description |
|------|-------------|
| `id` | The bucket ID |
| `global_alias` | The global alias |
| `bytes` | Current bucket size in bytes |
| `objects` | Current number of objects |
| `expiration_days` | Bucket expiration in days |

### garage_buckets

List all buckets in the cluster.

```hcl
data "garage_buckets" "all" {}
```

**Attributes:**

| Name | Description |
|------|-------------|
| `buckets` | List of all buckets |

### garage_cluster_status

Get the cluster status including node information, layout, and storage details.

```hcl
data "garage_cluster_status" "this" {}
```

**Attributes:**

| Name | Description |
|------|-------------|
| `nodes` | List of cluster nodes with status and capacity |
| `layout` | Current cluster layout version |
| `staging` | Staging layout changes |

### garage_cluster_health

Get cluster health metrics including connectivity and storage status.

```hcl
data "garage_cluster_health" "this" {}
```

**Attributes:**

| Name | Description |
|------|-------------|
| `status` | Overall cluster health status |
| `nodes` | Per-node health information |
| `storage` | Storage usage and capacity details |

## Installation

After building, install to your local Terraform plugins directory:

```bash
make install
```

This installs the provider to `~/.terraform.d/plugins/registry.terraform.io/d0ugal/garage/0.0.1/linux_amd64/`

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (to build the provider plugin)
- Access to a Garage instance with admin API enabled

## Development

```bash
# Build
make build

# Test
make test

# Lint
make lint

# Clean
make clean
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

