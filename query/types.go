package query

type MongoDBQuerySpec struct {
	Operation                    MongoDBOperation `json:"operation"`
	DatabaseName                 string           `json:"databaseName"`
	CollectionName               string           `json:"collectionName"`
	Command                      string           `json:"command"`
	Count                        *int64           `json:"count,omitempty"`
	AvgExecutionTimeMilliSeconds *int64           `json:"avgExecutionTimeMilliSeconds,omitempty"`
	MinExecutionTimeMilliSeconds *int64           `json:"minExecutionTimeMilliSeconds,omitempty"`
	MaxExecutionTimeMilliSeconds *int64           `json:"maxExecutionTimeMilliSeconds,omitempty"`
}

type MongoDBOperation string

const (
	MongoDBOperationQuery   MongoDBOperation = "QUERY"
	MongoDBOperationInsert  MongoDBOperation = "INSERT"
	MongoDBOperationUpdate  MongoDBOperation = "UPDATE"
	MongoDBOperationDelete  MongoDBOperation = "DELETE"
	MongoDBOperationGetMore MongoDBOperation = "GETMORE"
)

type QueryOps struct {
	Operation  string `json:"operation"`
	Database   string `json:"database"`
	Collection string `json:"collection"`
	// Ref: https://docs.mongodb.com/manual/reference/database-profiler/#mongodb-data-system.profile.command
	Command string `json:"command"`
}

type QueryExecInfo struct {
	Count int64 `json:"count"`
	Avg   int64 `json:"avg"`
	Min   int64 `json:"min"`
	Max   int64 `json:"max"`
}
