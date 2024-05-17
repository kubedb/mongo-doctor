package query

import (
	"bytes"
	"encoding/json"
	"errors"
	"go.mongodb.org/mongo-driver/mongo"
	"gomodules.xyz/pointer"
	"k8s.io/klog/v2"
	"kubedb.dev/mongo-doctor/database"
	"kubedb.dev/mongo-doctor/utils"
	"math"
	"sort"
	"strconv"
	"strings"
)

func Run(client *mongo.Client, fileName string) {
	utils.MakeDir(utils.Dir)
	queries, err := getQueriesFromPod(client)
	if err != nil {
		klog.Fatal(err)
	}
	sort.Slice(queries, func(i, j int) bool {
		return pointer.Int64(queries[i].AvgExecutionTimeMilliSeconds) > pointer.Int64(queries[j].AvgExecutionTimeMilliSeconds)
	})
	data, err := json.MarshalIndent(queries, "", "  ")
	if err != nil {
		klog.Fatal(err)
	}
	utils.WriteFile(utils.Dir, fileName, data)
}

func getQueriesFromPod(client *mongo.Client) ([]MongoDBQuerySpec, error) {
	dbs := database.ListDatabases(client)
	queryInfo := make(map[QueryOps]QueryExecInfo)

	for _, databaseName := range dbs {
		if findInSlice(utils.SkipDBList, databaseName) && databaseName != "admin" {
			continue
		}

		err := SetDatabaseProfilingLevel(client, databaseName)
		if err != nil {
			return nil, err
		}
		queries, err := GetQueries(client, databaseName)
		if err != nil {
			return nil, err
		}

		for _, q := range queries {
			ops := QueryOps{}
			opsType := q["op"].(string)
			profileNS := q["ns"].(string)
			commandInfo, ok := q["command"].(map[string]interface{})
			if !ok {
				return nil, errors.New("failed to parse MongoDB .system.profile.command information")
			}
			command, err := getCommand(commandInfo)
			if err != nil {
				return nil, err
			}
			profileDB, profileColl, err := ParseDBColl(profileNS)
			if err != nil {
				return nil, err
			}
			ops.Operation = opsType
			ops.Database = profileDB
			ops.Collection = profileColl
			ops.Command = command

			execTimeInfo, found := queryInfo[ops]
			if !found {
				execTimeInfo = QueryExecInfo{
					Count: 0,
					Avg:   0,
					Min:   math.MaxInt32,
					Max:   0,
				}
			}
			currExTime := q["millis"].(map[string]interface{})
			exTimeInMs, err := strconv.ParseInt(currExTime["$numberInt"].(string), 10, 64)
			if err != nil {
				return nil, err
			}

			execTimeInfo.Avg = (execTimeInfo.Count*execTimeInfo.Avg + exTimeInMs) / (execTimeInfo.Count + 1)
			execTimeInfo.Min = minINT64(execTimeInfo.Min, exTimeInMs)
			execTimeInfo.Max = maxINT64(execTimeInfo.Max, exTimeInMs)
			execTimeInfo.Count += 1

			queryInfo[ops] = execTimeInfo
		}
	}

	qList := make([]MongoDBQuerySpec, 0, len(queryInfo))
	for k, v := range queryInfo {
		if findInSlice(utils.SkipCollectionList, k.Collection) || findInSlice(utils.SkipDBList, k.Database) {
			continue
		}
		query := MongoDBQuerySpec{
			Operation:                    MongoDBOperation(k.Operation),
			DatabaseName:                 k.Database,
			CollectionName:               k.Collection,
			Command:                      k.Command,
			Count:                        pointer.Int64P(v.Count),
			AvgExecutionTimeMilliSeconds: pointer.Int64P(v.Avg),
			MinExecutionTimeMilliSeconds: pointer.Int64P(v.Min),
			MaxExecutionTimeMilliSeconds: pointer.Int64P(v.Max),
		}
		qList = append(qList, query)
	}

	return qList, nil
}

func ParseDBColl(profileNS string) (string, string, error) {
	res := strings.Split(profileNS, ".")
	if len(res) < 1 {
		return "", "", errors.New("profile namespace is corrupted")
	}
	profileDB := res[0]
	profileColl := ""
	for i := 1; i < len(res); i++ {
		profileColl += res[i]
		if i != len(res)-1 {
			profileColl += "."
		}
	}
	return profileDB, profileColl, nil
}

func getCommand(cmdInfo map[string]interface{}) (string, error) {
	if cmdInfo == nil {
		return "", nil
	}
	cmdData, err := json.Marshal(cmdInfo)
	if err != nil {
		return "", err
	}
	return bytes.NewBuffer(cmdData).String(), nil
}

func findInSlice(values []string, key string) bool {
	for _, v := range values {
		if v == key {
			return true
		}
	}
	return false
}

func maxINT64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func minINT64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
