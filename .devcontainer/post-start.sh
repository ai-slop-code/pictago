#!/usr/bin/env bash
set -euo pipefail

# Load environment variables
# shellcheck source=./config/terraform.env
if [[ -f "/home/vscode/.devcontainer/config/terraform.env" ]]; then
  echo "Loading Terraform environment variables..."
  set -a
  source "/home/vscode/.devcontainer/config/terraform.env"
  set +a
fi

# Make hack executable
chmod +x /home/vscode/.devcontainer/scripts/*.sh

# Display welcome message
clear
printf "\e[0;32mTerraform Development Environment: %s\e[0m\n\n" "$(basename "${PWD}")"

# Display installed tools and versions
echo "=== Installed Tools ==="
echo "Terraform: $(terraform --version | head -n 1)"
echo ""

# Display environment information
echo "=== Environment Information ==="
echo "Working Directory: $(pwd)"
echo "User: $(whoami)"
echo ""

# Display helpful commands
echo "=== Helpful Commands ==="
echo "terraform init - Initialize a Terraform working directory"
echo "terraform plan - Generate and show an execution plan"
echo "terraform apply - Builds or changes infrastructure"
echo "terraform validate - Validates the Terraform files"
echo "terraform fmt - Rewrites config files to canonical format"
echo ""

# Display container information if available
if command -v devcontainer-info &> /dev/null; then
  devcontainer-info
fi
