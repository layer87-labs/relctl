package jenkinspipeline

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServerURL(t *testing.T) {
	tests := []struct {
		name, explicit, gitURL, want string
	}{
		{"github.com https", "", "https://github.com/example-org/example-repo.git", "https://github.com"},
		{"github enterprise https", "", "https://ghe.example.com/example-org/example-repo.git", "https://ghe.example.com"},
		{"https with port", "", "https://ghe.example.com:8443/org/repo", "https://ghe.example.com"},
		{"scp-like ssh", "", "git@github.com:example-org/example-repo.git", "https://github.com"},
		{"scp-like ssh enterprise", "", "git@ghe.example.com:org/repo.git", "https://ghe.example.com"},
		{"ssh scheme", "", "ssh://git@ghe.example.com/org/repo.git", "https://ghe.example.com"},
		{"http stays http", "", "http://ghe.internal.example/org/repo.git", "http://ghe.internal.example"},
		{"explicit wins", "https://ghe.example.com/", "git@github.com:org/repo.git", "https://ghe.example.com"},
		{"unparsable is kept", "", "not a url", "not a url"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ServerURL(tt.explicit, tt.gitURL))
		})
	}
}
