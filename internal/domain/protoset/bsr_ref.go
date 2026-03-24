package protoset

import "strings"

func parseBSRModuleRef(raw string) (string, string, bool) {
	if raw == "" {
		return "", "", false
	}

	module, version := raw, ""

	slash := strings.IndexByte(raw, '/')
	if sep := strings.LastIndexAny(raw, "@:"); sep > 0 {
		if raw[sep] == '@' || sep > slash {
			module, version = raw[:sep], raw[sep+1:]
		}
	}

	parts := strings.Split(module, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", false
	}

	if !strings.Contains(parts[0], ".") {
		return "", "", false
	}

	return module, version, true
}
