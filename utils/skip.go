package utils

var (
	SkipDBList         = []string{"admin", "config", "local"}
	SkipCollectionList = []string{"system.profile", "system.js", "system.views"}
)

func SkipDB(dbname string) bool {
	for _, s := range SkipDBList {
		if dbname == s {
			return true
		}
	}
	return false
}

func SkipCollection(collectionName string) bool {
	for _, s := range SkipCollectionList {
		if collectionName == s {
			return true
		}
	}
	return false
}
