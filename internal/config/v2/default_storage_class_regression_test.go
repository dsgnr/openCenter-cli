package v2

import (
	"regexp"
	"testing"
)

func TestDefaultStorageClassMustBeDNS1123(t *testing.T) {
	dns1123 := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)

	cases := []struct {
		provider string
		region   string
	}{
		{"openstack", "iad3-staging"},
		{"vmware", "unknown-region"},
		{"kind", "any"},
		{"baremetal", "any"},
	}

	for _, tc := range cases {
		t.Run(tc.provider+"/"+tc.region, func(t *testing.T) {
			got := defaultStorageClass(tc.provider, tc.region)
			if !dns1123.MatchString(got) {
				t.Fatalf("defaultStorageClass(%q, %q) = %q, must be a lowercase RFC 1123 subdomain",
					tc.provider, tc.region, got)
			}
		})
	}
}
