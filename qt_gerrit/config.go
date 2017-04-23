package main

// This is based on a whitelist (for now). Feel free to add additional entries.

// Map a Gerrit project to a Github one.
var gerritToGithubMap = map[string]string{
	"qt/qt5":           "qt/qt5",
	"qt/qtbase":        "qt/qtbase",
	"qt/qtdeclarative": "qt/qtdeclarative",
}

// Map a bare repository name to a Github one.
var repoNameToGithubMap = map[string]string{
	"qt5":           "qt/qt5",
	"qtbase":        "qt/qtbase",
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
