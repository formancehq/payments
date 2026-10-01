// Command versionprobe prints the linked plugin version. It exists so the
// release-like SemVer injection is proven by a real link step rather than by
// reading the build script.
package main

import (
	"fmt"

	"github.com/formancehq/payments/plugins/fctl/core"
)

func main() {
	fmt.Print(core.Plugin{}.Metadata().Version)
}
