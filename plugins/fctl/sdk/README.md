# fctl SDK snapshot

`fctl-v2-poc/` holds a minimal Go SDK snapshot and the public plugin WIT copied
from `formancehq/fctl-v2-poc` commit
`e9b1395f46f3100b381dbe00f5213de28e6df0e1`. The production files it contains are
exactly the union of the fctl packages `go list -deps` reaches for this plugin
in its normal build and in its `fctl_component_guest` build, plus the Go module
manifests and the public WIT. SDK tests, test data, and TypeScript sources are
never consumed by this plugin and are omitted.

The source repository is private while Payments CI receives a token scoped to
the Payments repository, so a cross-repository credential is the one thing the
gates must not need. Keeping this snapshot in the tree makes them self-contained
instead.

`../fctl-sdk.lock.json` is the fail-closed provenance manifest.
`scripts/with-fctl-sdk.sh` verifies the module path, the exact Nix content hash
of this snapshot (`bundleNarHash`), and the WIT SHA-256 before any tidy, test,
or component build consumes it. Adding, removing, or editing a file here
therefore fails every gate until the manifest is resealed deliberately.

`FCTL_SDK_ROOT` still overrides this snapshot with a local SDK checkout. That
path is verified against `commit` and `sdkNarHash`, which cover the whole
upstream module rather than this measured subset.
