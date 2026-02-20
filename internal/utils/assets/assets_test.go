package assets

import "testing"

func TestIsValid_ValidAssets(t *testing.T) {
	cases := []string{
		"A",
		"BTC",
		"ETH_TEST",
		"ETH-AETH_SEPOLIA",
		"USD/2",
	}
	for _, c := range cases {
		c := c
		t.Run(c, func(t *testing.T) {
			if !IsValid(c) {
				t.Fatalf("expected %q to be valid", c)
			}
		})
	}
}

func TestIsValid_InvalidAssets(t *testing.T) {
	cases := []string{
		"",                   // empty
		"eth",                // lowercase not allowed
		"1ABC",               // must start with uppercase letter
		"ABC.DEF",            // dot not allowed
		"USD/",               // trailing slash without version
		"USD/ABC",            // non-digit version
		"USD/1234567",        // version too long (7 digits)
		"A12345678901234567", // 18 chars before slash
	}
	for _, c := range cases {
		c := c
		t.Run(c, func(t *testing.T) {
			if IsValid(c) {
				t.Fatalf("expected %q to be invalid", c)
			}
		})
	}
}
