package main

// Map a bare repository name to a Github one.
// Used for the text triggers, e.g. "look at commit qtbase/<sha>"
// This is based on a whitelist (for now). Feel free to add additional entries.
var repoNameToGithubMap = map[string]string{
	"qt5":           "qt/qt5",
	"qtdoc":         "qt/qtdoc",
	"qtbase":        "qt/qtbase",
	"qtmultimedia":  "qt/qtmultimedia",
	"qtdeclarative": "qt/qtdeclarative",
}

var validBareRepoNames []string

func init() {
	validBareRepoNames = make([]string, len(repoNameToGithubMap))
	i := 0
	for k := range repoNameToGithubMap {
		validBareRepoNames[i] = k
		i++
	}
}
