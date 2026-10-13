---
id: kind-local-development
title: Local development with Kind
sidebar_label: Local development with Kind
description: Configure, render, and deploy a Kind cluster for local openCenter development.
doc_type: tutorial
audience: openCenter developers
tags: [kind, local-development]
last_updated: 2026-10-05
---
# Local development with Kind

The Kind provider is the repository's local-provider path. This tutorial
covers the openCenter commands that create, render, and deploy a local Kind
cluster, ending with a running, Flux-managed cluster reconciled from a local
Gitea. It does not prove a particular Docker/Podman installation or service
set is present.

## Prerequisites

- A container runtime (`docker` or `podman`) with a running daemon, and
  `kind` + `kubectl` + `helm` + `flux` on your PATH (mise can install these).
- On Apple Silicon and other ARM hosts, Rosetta emulation for amd64
  containers. The `goharbor/*` images are published for amd64 only, and
  without Rosetta the Harbor Go services (core, exporter, jobservice,
  registry and Trivy) crash on start with `fatal error: lfstack.push`.
  In Docker Desktop, enable **Use Rosetta for x86_64/amd64 emulation on Apple
  Silicon**. With Colima, start the VM with `--vm-type vz --vz-rosetta`.
- A local Gitea for the GitOps repository. Start and provision it with:

  ```bash
  opencenter local gitea up
  opencenter local gitea status
  ```

  This creates the container, the `admin`/user tokens under
  `~/.config/opencenter/local/tokens/`, and the test repository that the
  cluster's GitOps `origin` points at.

## Build the CLI

From the repository root, build the binary with mise:

```bash
mise run build-cli        # -> ./bin/opencenter
```

Use `./bin/opencenter` in the commands below (or add `bin/` to your PATH).

## Create the configuration

```bash
opencenter cluster init dev --org local --type kind
```

If openCenter should manage the CNI instead of Kind's default CNI, use the
provider-specific flag:

```bash
opencenter cluster init dev --org local --type kind \
  --kind-disable-default-cni
```

The Kind init defaults the GitOps provider to **gitea** (local,
offline-friendly), disables OpenTofu, and clears OpenStack auth. Inspect the
generated paths and effective values:

```bash
opencenter cluster describe local/dev
```

## Point the GitOps repository at the local Gitea

`cluster init` writes placeholder GitOps values (a `github.com` URL and
`token: CHANGEME`). Replace them with your local Gitea before validating:

```bash
opencenter cluster set local/dev \
  "opencenter.gitops.repository.url=<git-url>" \
  "opencenter.gitops.auth.token.token_file=$HOME/.config/opencenter/local/tokens/gitea-user.token"
```

The GitOps URL host is added to the local Gitea's TLS certificate as a SAN
during the `gitea-attach-kind` deploy step (alongside `localhost`, `gitea`, the
Kind network IP, and the host's routable IP). So a friendly `/etc/hosts`
hostname (e.g. `gitea.oc-baremetal`) works with TLS — you do not have to use
the bare host IP. See
[Cert permissions and TLS hostname](#cert-permissions-and-tls-hostname) below.

The GitOps `origin` remote in the local checkout must match this URL; the
deploy pre-check `verifyOriginMatchesGitURL` rejects a mismatch:

```bash
git -C <gitops-local-dir> remote set-url origin <git-url>
```

## Make the configuration valid

`cluster validate` gates the deploy. A fresh Kind init commonly fails on:

- **Placeholder secrets** — `secrets.loki.*`, `secrets.tempo.*`,
  `secrets.keycloak.admin_password`, etc. are `CHANGEME` until set.
- **Missing S3 endpoints** — when Loki/Tempo are enabled, set
  `opencenter.services.loki.s3_endpoint` and
  `opencenter.services.tempo.s3_endpoint` to a reachable object store
  (e.g. a local RustFS: `http://<host-ip>:9000`).
- **Schedulability** — `opencenter.services.keycloak.instances` must not
  exceed the number of schedulable Linux workers
  (`opencenter.infrastructure.kind.worker_count`).

Set values with `cluster set` (dot notation), then re-validate until it
reports `passed` with `0 failed`:

```bash
opencenter cluster validate local/dev
```

## Generate, sync secrets, and commit

```bash
opencenter cluster generate local/dev      # render GitOps manifests
opencenter secrets sync local/dev          # encrypt + wire secret manifests
```

`cluster generate` refuses to overwrite tracked files in the shared GitOps
working tree, and the deploy pre-check requires the tree to be clean with an
`origin` that matches `git_url`. If a previous cluster left the tree dirty
(modified tracked files or stray terraform state), restore and clean it first:

```bash
git -C <gitops-local-dir> checkout -- .gitignore .sops.yaml   # tracked files
git -C <gitops-local-dir> status --porcelain                  # remove/commit the rest
git -C <gitops-local-dir> add -A
git -C <gitops-local-dir> commit -m "add <cluster> gitops manifests"
```

Generation can be previewed with the global `--dry-run`, or limited to
template rendering with `--render-only`.

## Deploy with the selected container runtime

```bash
opencenter cluster deploy local/dev --container-runtime podman
```

The deploy flag accepts `docker` or `podman`. The provider implementation uses
the selected runtime for Kind operations; the repository does not establish
that either runtime is installed on a user's machine.

For a preview:

```bash
opencenter --dry-run cluster deploy local/dev --container-runtime podman
```

The deploy runs these ordered steps (reported by the plan, not hard-coded
here):

1. `kind-create`
2. `kind-export-kubeconfig`
3. `kind-install-cni` (managed-CNI mode only)
4. `gitea-attach-kind`
5. `flux-bootstrap`
6. `reconcile-sops-age-secret`
7. `reconcile-grafana-admin-secret`
8. `gitea-rebase`
9. `gitops-push`
10. `flux-verify`

If a step fails, rerun deploy to resume from saved state. `--restart` ignores
saved state and `--from-step <id>` starts at a named step. Use `--break-lock`
to clear a stale operation lock.

## Cert permissions and TLS hostname

`gitea-attach-kind` reissues the Gitea TLS certificate and attaches Gitea to the
Kind network. Three historical failure modes are now handled by the CLI; the
manual steps below are fallbacks only for older builds.

- **Cert file ownership (handled).** The data dir may be owned by the container
  user (UID 1000) while the CLI runs as a different user. `writeCertificates`
  drops foreign-owned cert files (which only needs directory write) and
  re-creates them, then re-owns the certs to the container UID so the Gitea
  process can read `key.pem`. If your build predates this and the step fails
  with `permission denied`, grant both users access and retry:

  ```bash
  CERTS=~/.config/opencenter/local/gitea/gitea/certs
  sudo chown 1000:1000 "$CERTS"/ca.pem "$CERTS"/cert.pem "$CERTS"/key.pem
  sudo setfacl -m u:$(id -u):rw "$CERTS"/ca.pem "$CERTS"/cert.pem "$CERTS"/key.pem
  podman restart gitea   # if it is down (could not load key.pem)
  ```

- **TLS hostname (handled).** The reissued cert includes the GitOps URL's host
  as a DNS SAN in addition to `localhost`, `gitea`, the Kind network IP, and the
  host routable IP. So a friendly `/etc/hosts` name (e.g. `gitea.oc-baremetal`)
  in `opencenter.gitops.repository.url` works with TLS — no need to switch the
  URL to the bare host IP. Verify with the CA before deploying:

  ```bash
  curl --cacert ~/.config/opencenter/local/gitea/gitea/certs/ca.pem \
    https://<git-host>:3001/ -o /dev/null -w '%{http_code}\n'   # expect 200
  ```

- **Re-generating after a URL change (handled).** `cluster generate` no longer
  refuses on `flux bootstrap`'s `gotk-*` files under
  `applications/overlays/<cluster>/flux-system/` (they are deploy-managed, not
  generator-owned). If you change `opencenter.gitops.repository.url` after
  generating, just re-run `cluster generate` — it rewrites the
  `opencenter-keycloak-config` / `opencenter-olm-config` sources and the
  `gotk-*` files are left alone. Commit, then push with
  `opencenter cluster deploy <cluster> --step gitops-push`.

## Inspect the resulting cluster

Use the kubeconfig path shown by `cluster describe` or the deploy output
(`~/.config/opencenter/clusters/state/<org>/<cluster>/kubeconfig.yaml`) for
live inspection:

```bash
export KUBECONFIG=~/.config/opencenter/clusters/state/local/dev/kubeconfig.yaml
kubectl get nodes
kubectl get kustomizations -A
```

A successful deploy reports `Cluster ready at https://127.0.0.1:6443`.
Confirm with the CLI:

```bash
opencenter cluster status local/dev --refresh
```

`Status: success` with a ready API endpoint and Ready nodes is the end state.
Flux's `GitRepository` points at the local Gitea and the `flux-system`
kustomization is `True`, confirming the cluster is reconciled from the local
repository. These checks require a running local cluster and tools outside
this repository's workflow fixture.

## Evidence

- Kind flags: `cmd/cluster_init.go`, `cmd/cluster_deploy.go`
- Provider validation: `internal/config/v2/readiness.go`
- Kind bootstrap implementation: `internal/cluster/kind_bootstrap_provider.go`
- Local Gitea attach + cert reissue: `internal/localdev/gitea/service.go`
- GitOps origin pre-check: `cmd/cluster_deploy.go` (`verifyOriginMatchesGitURL`)
- Local Gitea lifecycle: `cmd/opencenter-local` (`gitea up|status|attach-kind|destroy`)
- Workflow limitation: `tests/features/workflow.feature:1-8`
