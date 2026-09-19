package version

import "testing"

func TestModuleVersion(t *testing.T) {
	tests := map[string]string{
		"v1.0.5":                             "1.0.5",
		"v1.2.3-rc.1":                        "1.2.3-rc.1",
		"v0.0.0-20260818114237-f8c1a4eacefd": "0.0.0-20260818114237-f8c1a4eacefd",
		"(devel)":                            "devel",
		"":                                   "devel",
	}

	for input, want := range tests {
		if got := moduleVersion(input); got != want {
			t.Errorf("moduleVersion(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestAppVersion(t *testing.T) {
	if version := Version(); version == "" {
		t.Errorf("Version() should not be blank")
	}
}
