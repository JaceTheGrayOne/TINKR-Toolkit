package utils

import "strings"

// FormatDisplayName converts a mod folder name into a human-readable display name.
// Removes z_ prefix, strips numeric codes like _0001_P, and replaces underscores with spaces.
func FormatDisplayName(folderName string) string {
	name := strings.TrimPrefix(folderName, "z_")
	name = strings.TrimPrefix(name, "Z_")

	if idx := strings.LastIndex(name, "_"); idx != -1 {
		if strings.HasSuffix(name, "_P") {
			parts := strings.Split(name, "_")
			if len(parts) >= 2 {
				secondLast := parts[len(parts)-2]
				if len(secondLast) == 4 {
					allDigits := true
					for _, c := range secondLast {
						if c < '0' || c > '9' {
							allDigits = false
							break
						}
					}
					if allDigits {
						name = strings.Join(parts[:len(parts)-2], "_")
					}
				}
			}
		}
	}

	name = strings.ReplaceAll(name, "_", " ")
	return name
}
