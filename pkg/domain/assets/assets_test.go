package assets

import "testing"

func TestIsValid_ValidAssets(t *testing.T) {
	cases := []string{
		"A",
		"BTC",
		"USD",
		"USD/2",
		"USD/1234",
		"EUR/00",
		"USD123",
		"EUR_COL",
		"EUR_COL/12",
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
		"ETH_TEST5",          // digit in suffix segment
		"ETH-AETH_SEPOLIA",   // hyphen not allowed
		"MATIC_POLYGON_MUMBAI", // multiple underscore segments
		"EUR_",               // empty suffix segment
		"_C",                 // leading underscore
		"A_/2",               // empty suffix segment before /
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
