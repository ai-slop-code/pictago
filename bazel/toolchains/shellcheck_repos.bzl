_PLATFORMS = {
    ("linux", "amd64"): (
        "https://github.com/koalaman/shellcheck/releases/download/v0.11.0/shellcheck-v0.11.0.linux.x86_64.tar.xz",
        "8c3be12b05d5c177a04c29e3c78ce89ac86f1595681cab149b65b97c4e227198",
        "shellcheck-v0.11.0",
    ),
    ("linux", "arm64"): (
        "https://github.com/koalaman/shellcheck/releases/download/v0.11.0/shellcheck-v0.11.0.linux.aarch64.tar.xz",
        "12b331c1d2db6b9eb13cfca64306b1b157a86eb69db83023e261eaa7e7c14588",
        "shellcheck-v0.11.0",
    ),
    ("darwin", "amd64"): (
        "https://github.com/koalaman/shellcheck/releases/download/v0.11.0/shellcheck-v0.11.0.darwin.x86_64.tar.xz",
        "3c89db4edcab7cf1c27bff178882e0f6f27f7afdf54e859fa041fca10febe4c6",
        "shellcheck-v0.11.0",
    ),
    ("darwin", "arm64"): (
        "https://github.com/koalaman/shellcheck/releases/download/v0.11.0/shellcheck-v0.11.0.darwin.aarch64.tar.xz",
        "56affdd8de5527894dca6dc3d7e0a99a873b0f004d7aabc30ae407d3f48b0a79",
        "shellcheck-v0.11.0",
    ),
}

def _shellcheck_repo_impl(ctx):
    url = ctx.attr.url
    sha256 = ctx.attr.sha256
    strip_prefix = ctx.attr.strip_prefix

    ctx.download(
        url = url,
        sha256 = sha256,
        output = "shellcheck.tar.xz",
    )
    ctx.extract(archive = "shellcheck.tar.xz", strip_prefix = strip_prefix)
    ctx.file(
        "BUILD.bazel",
        'exports_files(["shellcheck"], visibility = ["//visibility:public"])',
    )

_shellcheck_repo = repository_rule(
    implementation = _shellcheck_repo_impl,
    attrs = {
        "sha256": attr.string(mandatory = True),
        "strip_prefix": attr.string(mandatory = True),
        "url": attr.string(mandatory = True),
    },
)

def _shellcheck_repos_impl(ctx):
    # Only process if there are download tags
    if not ctx.modules[0].tags.download:
        return

    # Create a repository for each platform
    for platform_key, platform_info in _PLATFORMS.items():
        platform_os = platform_key[0]
        platform_arch = platform_key[1]
        url = platform_info[0]
        sha256 = platform_info[1]
        strip_prefix = platform_info[2]

        repo_name = "com_github_koalaman_shellcheck_{}_{}".format(platform_os, platform_arch)
        _shellcheck_repo(
            name = repo_name,
            url = url,
            sha256 = sha256,
            strip_prefix = strip_prefix,
        )

shellcheck_repos = module_extension(
    implementation = _shellcheck_repos_impl,
    tag_classes = {
        "download": tag_class(attrs = {}),
    },
)
