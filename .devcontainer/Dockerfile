ARG VARIANT="jammy"
FROM mcr.microsoft.com/vscode/devcontainers/base:0-${VARIANT}

# Set versions as arguments
ARG TERRAFORM_VERSION="1.13.3"
ARG TERRAFORM_DOCS_VERSION="0.20.0"

# Copy library hack
COPY scripts/*.sh /tmp/scripts/
RUN chmod +x /tmp/hack/*.sh

# Install common utilities and dependencies
USER root
RUN /tmp/hack/common-utils.sh

# Install Terraform and related tools
RUN /tmp/hack/terraform-tools.sh \
    "${TERRAFORM_VERSION}" \
    "${TERRAFORM_DOCS_VERSION}"

COPY config /home/vscode/.devcontainer/config
RUN chown -R vscode:vscode /home/vscode/.devcontainer

# Set up Terraform plugin cache directory
RUN mkdir -p /home/vscode/.terraform.d/plugin-cache \
    && chown -R vscode:vscode /home/vscode/.terraform.d

# Set up credential directories
RUN mkdir -p /home/vscode/.aws \
    && mkdir -p /home/vscode/.azure \
    && mkdir -p /home/vscode/.config/gcloud \
    && mkdir -p /home/vscode/.ssh \
    && chown -R vscode:vscode /home/vscode/.aws \
    && chown -R vscode:vscode /home/vscode/.azure \
    && chown -R vscode:vscode /home/vscode/.config \
    && chown -R vscode:vscode /home/vscode/.ssh \
    && chmod 700 /home/vscode/.ssh

# Clean up
RUN apt-get clean -y \
    && rm -rf /var/lib/apt/lists/* \
    && rm -rf /tmp/hack

# Copy post-start script
COPY post-start.sh /usr/local/bin/post-start
RUN chmod +x /usr/local/bin/post-start

# Switch back to vscode user
USER vscode
