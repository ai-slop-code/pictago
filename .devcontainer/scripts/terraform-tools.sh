#!/usr/bin/env bash
set -euo pipefail

# This script installs Terraform and related tools

# Versions
TERRAFORM_VERSION=${1:-"1.13.3"}
TERRAFORM_DOCS_VERSION=${2:-"0.20.0"}

echo "Installing Terraform v${TERRAFORM_VERSION}..."
curl -sSL -o /tmp/terraform.zip "https://releases.hashicorp.com/terraform/${TERRAFORM_VERSION}/terraform_${TERRAFORM_VERSION}_linux_amd64.zip"
unzip -qq /tmp/terraform.zip -d /tmp
sudo mv /tmp/terraform /usr/local/bin/
rm -f /tmp/terraform.zip

echo "Installing terraform-docs v${TERRAFORM_DOCS_VERSION}..."
curl -sSLo /tmp/terraform-docs.tar.gz "https://github.com/terraform-docs/terraform-docs/releases/download/v${TERRAFORM_DOCS_VERSION}/terraform-docs-v${TERRAFORM_DOCS_VERSION}-linux-amd64.tar.gz"
tar -xzf /tmp/terraform-docs.tar.gz -C /tmp
sudo mv /tmp/terraform-docs /usr/local/bin/
rm -f /tmp/terraform-docs.tar.gz

echo "Terraform tools installation complete!"
