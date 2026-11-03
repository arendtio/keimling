package main

// buildVersion is the short git commit this binary was built from.
// Builds that do not stamp it stay "dev".
var buildVersion = "dev"

func versionLine() string {
	return "keimling " + buildVersion
}
