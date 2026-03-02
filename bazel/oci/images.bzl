"""Reusable container image configuration for dataflow_exporter.

This module provides a reusable way to build container images matching the
original Dockerfile structure. It can be imported and used in MODULE.bazel.

Usage in BUILD.bazel:
    load("//container:images.bzl", "declare_container_images")

    declare_container_images(
        name = "dynatrace_reporter_images",
        base_image = "@ubuntu_2404",
        binary = "//cmd/dataflow_exporter:dataflow_exporter",
        entrypoint = "//hack:entrypoint.sh",
    )
"""

load("@rules_oci//oci:defs.bzl", "oci_image", "oci_load", "oci_push")
load("@rules_pkg//pkg:tar.bzl", "pkg_tar")

def declare_container_images(
        name,
        base_image,
        binary,
        entrypoint,
        registry = None,
        repository = None,
        user = "65532:65532",
        workdir = "/home/nonroot",
        env = None):
    """Declares container image build targets for the application.

    This function creates a complete set of container image targets matching
    the production Dockerfile, including:
    - oci_image: The container image definition
    - oci_load: Load image into local Docker
    - oci_push: Push image to registry

    Args:
        name: Base name for the targets (e.g., "app")
        base_image: Label of the base OCI image (e.g., "@ubuntu_2404")
        binary: Label of the Go binary target (e.g., "//cmd/app:app")
        entrypoint: Label of the entrypoint script (e.g., "//hack:entrypoint.sh")
        registry: Container registry (e.g., "ghcr.io")
        repository: Repository path (e.g., "your-org/dataflow_exporter")
        user: User/UID to run as (default: "65532:65532")
        workdir: Working directory (default: "/home/nonroot")
        env: Dictionary of environment variables (default: TERM and HOME)

    Targets created:
        :{name}_app_layer - Binary layer
        :{name}_entrypoint_layer - Entrypoint layer
        :{name} - The OCI image
        :{name}_load - Load into Docker
        :{name}_push - Push to registry (if registry/repository specified)
    """

    if env == None:
        env = {
            "TERM": "xterm",
            "HOME": workdir,
        }

    # Package the Go binary
    pkg_tar(
        name = "{}_app_layer".format(name),
        srcs = [binary],
        package_dir = "/bin",
        mode = "0755",
        visibility = ["//visibility:public"],
    )

    # Package the entrypoint script
    pkg_tar(
        name = "{}_entrypoint_layer".format(name),
        srcs = [entrypoint],
        package_dir = "/usr/local/bin",
        mode = "0755",
        remap_paths = {
            entrypoint.lstrip("/"): "/usr/local/bin/entrypoint",
        },
        visibility = ["//visibility:public"],
    )

    # Create the production container image
    oci_image(
        name = name,
        base = base_image,
        tars = [
            ":{}_app_layer".format(name),
            ":{}_entrypoint_layer".format(name),
        ],
        entrypoint = ["/usr/local/bin/entrypoint"],
        env = env,
        user = user,
        workdir = workdir,
        visibility = ["//visibility:public"],
    )

    # Load the image for local docker usage
    oci_load(
        name = "{}_load".format(name),
        image = ":{}".format(name),
        repo_tags = ["{}:latest".format(name)],
    )

    # Push target for publishing the image
    if registry and repository:
        oci_push(
            name = "{}_push".format(name),
            image = ":{}".format(name),
            repository = "{}/{}".format(registry, repository),
        )

