// Package sovereign exposes the reviewed declarative inputs compiled into suitectl.
//
// Release binaries must work without a Git checkout, so the catalog, profiles and
// the installation allowlist are embedded at build time. The allowlist is only
// ever read from this embedded copy: pointing --root at another directory can
// change what is planned, never what may be installed.
package sovereign

import "embed"

//go:embed catalog/apps/*.yaml profiles/*.yaml catalog/install-allowlist.yaml
var Files embed.FS

// AllowlistPath is the embedded location of the reviewed installation targets.
const AllowlistPath = "catalog/install-allowlist.yaml"
