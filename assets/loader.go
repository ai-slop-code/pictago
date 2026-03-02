package assets

import "embed"

const ConfigFileName = "config.toml"

//go:embed config.toml
var CfgFs embed.FS
