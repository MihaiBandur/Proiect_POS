package dto

import (
	"net/netip"
	"path"
	"regexp"
	"strings"

	"job-management-service/internal/httpx"
)

func requireText(v *httpx.Validation, field string, value *string) {
	switch {
	case value == nil:
		v.Missing(field)
	case strings.TrimSpace(*value) == "":
		v.Invalid(field, "nu poate fi gol")
	}
}

func requireAbsolutePath(v *httpx.Validation, field string, value *string) {
	switch {
	case value == nil:
		v.Missing(field)
	case !path.IsAbs(*value) || path.Clean(*value) != *value:
		v.Invalid(field, "trebuie sa fie o cale absoluta simplificata, ex. /opt/scripts/backup.sh")
	}
}

func canonicalIP(value string) (string, bool) {
	addr, err := netip.ParseAddr(value)
	if err != nil || addr.Zone() != "" {
		return "", false
	}

	return addr.Unmap().String(), true
}

var hostnameLabel = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)

func isHostname(value string) bool {
	if len(value) == 0 || len(value) > 253 {
		return false
	}

	for label := range strings.SplitSeq(value, ".") {
		if !hostnameLabel.MatchString(label) {
			return false
		}
	}

	return true
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
