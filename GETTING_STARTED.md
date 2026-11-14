# Getting Started — Tool Installation Guide

Install all required tools before attending the **Container Security on Cloud Run** workshop.
Estimated setup time: **15–20 minutes**.

> [!NOTE]
> All commands in the workshop use a standard terminal (bash/zsh).
> Windows users should use **WSL 2** (Ubuntu) or **Git Bash** for compatibility.

---

## Required Tools

| Tool | Purpose | Min Version |
|---|---|---|
| [Google Cloud CLI (`gcloud`)](#1-google-cloud-cli-gcloud) | Deploy and manage GCP resources | 460+ |
| [Docker](#2-docker) | Build and run container images locally | 24+ |
| [Terraform](#3-terraform) | Provision GCP infrastructure as code | 1.9+ |
| [Trivy](#4-trivy) | Scan container images for CVEs | 0.55+ |
| [Go & govulncheck](#5-go--govulncheck) | Run deterministic Go code vulnerability analysis | 1.22+ |
| [curl](#6-curl) | Make HTTP requests to the live service | any |
| [jq](#7-jq) | Parse and extract JSON output in the terminal | 1.6+ |

---

## 1. Google Cloud CLI (`gcloud`)

Official guide: [Google Cloud SDK Installation Docs](https://cloud.google.com/sdk/docs/install)

### macOS
```bash
brew install --cask google-cloud-sdk
```

### Linux / Windows (WSL 2)
```bash
# Recommended official install script (works across Debian/Ubuntu, Fedora, Arch)
curl https://sdk.cloud.google.com | bash
exec -l $SHELL
```
*Distribution packages are also available:*
- **Arch Linux:** `sudo pacman -S google-cloud-cli` (or via AUR: `google-cloud-cli`)
- **Fedora / RHEL:** Follow the [official DNF/YUM repository setup](https://cloud.google.com/sdk/docs/install#rpm)
- **Debian / Ubuntu:** Follow the [official APT repository setup](https://cloud.google.com/sdk/docs/install#deb)

### Post-install & Verification
```bash
gcloud version
gcloud auth login
```

---

## 2. Docker

Official guide: [Docker Desktop & Engine Installation Docs](https://docs.docker.com/engine/install/)

### macOS
Download and install [Docker Desktop for Mac](https://docs.docker.com/desktop/install/mac-install/).

> [!TIP]
> On **Apple Silicon (M1/M2/M3/M4)**, always build with `--platform linux/amd64`
> when targeting Cloud Run, which runs on `amd64`:
> ```bash
> docker build --platform linux/amd64 -t myapp .
> ```

### Linux
Follow the official distribution-specific guide:
- **Ubuntu:** [Install Docker Engine on Ubuntu](https://docs.docker.com/engine/install/ubuntu/)
- **Debian:** [Install Docker Engine on Debian](https://docs.docker.com/engine/install/debian/)
- **Fedora:** [Install Docker Engine on Fedora](https://docs.docker.com/engine/install/fedora/)
- **Arch Linux:**
  ```bash
  sudo pacman -S docker docker-buildx
  sudo systemctl enable --now docker
  ```

*Remember to follow the [Linux post-installation steps](https://docs.docker.com/engine/install/linux-postinstall/) to manage Docker as a non-root user:*
```bash
sudo usermod -aG docker $USER
# Log out and log back in, or run: newgrp docker
```

### Windows
Install [Docker Desktop for Windows](https://docs.docker.com/desktop/install/windows-install/) with the **WSL 2 backend** enabled.

### Verification
```bash
docker version
docker run hello-world
```

---

## 3. Terraform

Official guide: [HashiCorp Terraform Installation Docs](https://developer.hashicorp.com/terraform/install)

### macOS
```bash
brew tap hashicorp/tap
brew install hashicorp/tap/terraform
```

### Linux / Windows (WSL 2)
- **Arch Linux:**
  ```bash
  sudo pacman -S terraform
  ```
- **Fedora:**
  ```bash
  sudo dnf install -y dnf-plugins-core
  sudo dnf config-manager --add-repo https://rpm.releases.hashicorp.com/fedora/hashicorp.repo
  sudo dnf install -y terraform
  ```
- **Ubuntu / Debian:** Follow the [HashiCorp Linux installation instructions](https://developer.hashicorp.com/terraform/install#linux).

### Verification
```bash
terraform version
```

---

## 4. Trivy

Official guide: [Aqua Security Trivy Installation Docs](https://trivy.dev/latest/getting-started/installation/)

### macOS
```bash
brew install aquasecurity/trivy/trivy
```

### Linux / Windows (WSL 2)
- **Arch Linux:**
  ```bash
  sudo pacman -S trivy
  ```
- **Fedora / RHEL / CentOS:**
  ```bash
  sudo rpm -ivh https://github.com/aquasecurity/trivy/releases/download/v0.58.0/trivy_0.58.0_Linux-64bit.rpm
  ```
  *(Or see the [official Trivy RPM repository guide](https://trivy.dev/latest/getting-started/installation/#rhelcentos))*
- **Debian / Ubuntu:** Follow the [official Trivy APT installation instructions](https://trivy.dev/latest/getting-started/installation/#package-manager).

### Verification & Quick Test
```bash
trivy version

# Optional: warm up the vulnerability database cache
trivy image us-central1-docker.pkg.dev/devfest-cloud-sby-2026-csec-run/todo-repo/todo-service:stable
```

---

## 5. Go & govulncheck

Official guide: [go.dev/doc/install](https://go.dev/doc/install) and [pkg.go.dev/golang.org/x/vuln/cmd/govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)

Used to deterministically audit application Go source code for called vulnerable functions.

### Installation
- **macOS:** `brew install go`
- **Arch Linux:** `sudo pacman -S go`
- **Fedora:** `sudo dnf install -y golang`
- **Ubuntu / Debian:** `sudo apt-get install -y golang` or follow [official Go tarball install](https://go.dev/doc/install)

### Install govulncheck
```bash
go install golang.org/x/vuln/cmd/govulncheck@latest

# Verify (ensure ~/go/bin is in your PATH)
govulncheck -version
```

---

## 6. curl

Pre-installed on macOS, WSL 2, and most Linux distributions.

### Linux / Windows (WSL 2)
```bash
# Ubuntu / Debian
sudo apt-get install -y curl

# Fedora
sudo dnf install -y curl

# Arch Linux
sudo pacman -S curl
```

---

## 7. jq

Official site: [jqlang.github.io/jq](https://jqlang.github.io/jq/download/)

### macOS
```bash
brew install jq
```

### Linux / Windows (WSL 2)
```bash
# Ubuntu / Debian
sudo apt-get install -y jq

# Fedora
sudo dnf install -y jq

# Arch Linux
sudo pacman -S jq
```

### Verification
```bash
jq --version
```

---

## Verify Everything

Run this checklist before the workshop:

```bash
gcloud version
docker version
terraform version
trivy version
go version
govulncheck -version
curl --version
jq --version
```

All commands should return a version string without errors.

---

## GCP Account Setup

You will need a Google Cloud account with billing enabled.

> [!IMPORTANT]
> New GCP accounts receive **$300 free credits**. The workshop resources cost
> approximately **$0.01–$0.05** total if cleaned up at the end. Use the cleanup
> script to avoid any ongoing charges.

```bash
# Log in
gcloud auth login
gcloud auth application-default login

# Set your project (provided by the workshop or your own)
export PROJECT_ID="your-project-id"
gcloud config set project $PROJECT_ID
gcloud auth application-default set-quota-project $PROJECT_ID
```

---

## Clone the Repository

```bash
git clone https://github.com/fzl-22/devfest-cloud-sby-2026-csec-run.git
cd devfest-cloud-sby-2026-csec-run
```

You're ready. Head back to [README.md](./README.md) to start the workshop. 🚀
