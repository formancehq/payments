package main

import (
	"os/exec"
	"testing"
)

func TestPluginVersionValidatorImplementsStrictSemVer2(t *testing.T) {
	valid := []string{"0.0.0", "1.2.3", "1.2.3-alpha", "1.2.3-alpha.1", "1.2.3+build.7", "1.2.3-0.3.7+sha.5114f85"}
	invalid := []string{"", "v1.2.3", "01.2.3", "1.02.3", "1.2.03", "1.2.3-01", "1.2.3-a..b", "1.2.3-.", "1.2.3+foo..bar"}
	for _, value := range valid {
		if output, err := exec.Command("bash", "validate-plugin-version.sh", value).CombinedOutput(); err != nil {
			t.Errorf("valid %q rejected: %v: %s", value, err, output)
		}
	}
	for _, value := range invalid {
		if err := exec.Command("bash", "validate-plugin-version.sh", value).Run(); err == nil {
			t.Errorf("invalid %q accepted", value)
		}
	}
}
