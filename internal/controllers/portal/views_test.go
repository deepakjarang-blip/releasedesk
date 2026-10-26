package portal

import (
	"net/http/httptest"
	"testing"
)

func TestPortalBasePath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/", want: ""},
		{path: "/example-app", want: ""},
		{path: "/portal", want: "/portal"},
		{path: "/portal/", want: "/portal"},
		{path: "/portal/example-app", want: "/portal"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			request := httptest.NewRequest("GET", test.path, nil)
			if got := portalBasePath(request); got != test.want {
				t.Errorf("portalBasePath(%q) = %q, want %q", test.path, got, test.want)
			}
		})
	}
}
