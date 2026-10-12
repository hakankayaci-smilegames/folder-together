//go:build !windows && !linux

package tunnel

import "errors"

func extractEmbeddedCloudflared() (string, error) {
	return "", errors.New("embedded cloudflared not available for this OS")
}
