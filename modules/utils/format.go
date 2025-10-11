package utils

import "strings"

// Normalize directory name
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
