package config

// DefaultPackageFlags contains the built-in default flags for each package manager
// These are used when the user doesn't specify their own defaults
var DefaultPackageFlags = map[string][]string{
	// APT defaults: non-interactive and don't install recommended packages
	"apt": {"-y", "--no-install-recommends"},
	
	// Snap defaults: empty - snaps are interactive by design and most don't need special flags
	"snap": {},
	
	// Flatpak defaults: install system-wide and assume yes for prompts
	"flatpak": {"--system", "--assumeyes"},
}

// GetDefaultFlags returns the default flags for a package manager
func GetDefaultFlags(manager string) []string {
	if flags, exists := DefaultPackageFlags[manager]; exists {
		// Return a copy to avoid accidental modification
		result := make([]string, len(flags))
		copy(result, flags)
		return result
	}
	return []string{}
}

// HasDefaultFlags checks if a package manager has default flags defined
func HasDefaultFlags(manager string) bool {
	_, exists := DefaultPackageFlags[manager]
	return exists
}

// GetSupportedPackageManagers returns all supported package managers
func GetSupportedPackageManagers() []string {
	managers := make([]string, 0, len(DefaultPackageFlags))
	for manager := range DefaultPackageFlags {
		managers = append(managers, manager)
	}
	return managers
}