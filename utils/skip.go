package utils

func SkipDB(dbname string) bool {
	return dbname == "admin" ||
		dbname == "config" ||
		dbname == "local"
}

func SkipCollection(collectionName string) bool {
	return collectionName == "system.profile" ||
		collectionName == "system.js" ||
		collectionName == "system.views"
}
