# Helmfire Examples

This directory contains example configurations for testing helmfire.

## Simple App

The `simple-app` directory contains a basic helmfile configuration with two releases:
- nginx (bitnami/nginx chart)
- redis (bitnami/redis chart)

### Try it out:

```bash
cd examples/simple-app

# Basic sync (requires helm and kubectl)
helmfire sync -f helmfile.yaml --dry-run

# With chart substitution
helmfire chart bitnami/nginx ../local-chart/my-nginx-chart
helmfire sync -f helmfile.yaml --dry-run

# With image substitution
helmfire image docker.io/bitnami/nginx:1.25.3 nginx:latest
helmfire sync -f helmfile.yaml --dry-run
```

## Local Chart

The `local-chart` directory contains a custom nginx chart that can be used to test chart substitution:

```bash
# Substitute the bitnami/nginx chart with our local chart
helmfire chart bitnami/nginx ./examples/local-chart/my-nginx-chart

# List active substitutions
helmfire list charts

# Run sync to apply
cd examples/simple-app
helmfire sync -f helmfile.yaml --dry-run
```

## Testing Image Substitution

Test image substitution without modifying helmfile or values:

```bash
cd examples/simple-app

# Add image substitution
helmfire image docker.io/bitnami/nginx:1.25.3 myregistry.io/nginx:custom

# List substitutions
helmfire list images

# Sync with substitution (dry-run to see the changes)
helmfire sync -f helmfile.yaml --dry-run
```

## Notes

- All examples use `--dry-run` by default to avoid actual cluster changes
- Remove `--dry-run` to actually deploy to your Kubernetes cluster
- Ensure you have `helm` and `kubectl` installed and configured
