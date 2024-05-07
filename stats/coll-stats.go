package stats

import (
	"context"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/klog/v2"
	"kubedb.dev/mongo-doctor/utils"
	"log"
	"path/filepath"
	"strings"
)

func collectionStats(db *mongo.Database, coll string) {
	cmd := bson.D{{"collStats", coll}, {"scale", 1048576}}
	var result bson.M
	err := db.RunCommand(context.TODO(), cmd).Decode(&result)
	if err != nil {
		if strings.Contains(err.Error(), "is a view, not a collection") {
			klog.Infoln(err.Error())
			return
		} else {
			log.Fatal(err)
		}
	}

	//var b []byte
	//buf := bytes.NewBuffer(b)
	//enc := json.NewEncoder(buf)
	//err = enc.Encode(&result)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//writeFile(fmt.Sprintf("%s.%s", db.Name(), coll), buf.Bytes())

	indentedData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	utils.WriteFile(filepath.Join(dir, db.Name()), coll, indentedData)
	specificFields(db, result)
}

/*
{
  ns: 'cc.one',
  count: 5500000,
  size: 241,
  avgObjSize: 46,
  storageSize: 152,
  wiredTiger.cache: {
      'bytes currently in the cache': 8699527,
      'bytes dirty in the cache cumulative': 6824354,
      'bytes read into cache': 2243540,
      'bytes written from cache': 378867011,
  },
  nindexes: 1,
  totalIndexSize: 150,
  totalSize: 303,
  indexSizes: { _id_: 150 },
  scaleFactor: 1048576,
}
*/

func specificFields(db *mongo.Database, result bson.M) {
	output := make(bson.M)
	for s, obj := range result {
		switch s {
		case "ns", "count", "size", "avgObjSize", "storageSize", "nindexes", "totalIndexSize", "totalSize", "indexSizes", "scaleFactor":
			output[s] = obj
		case "wiredTiger":
			typedObj := obj.(bson.M)["cache"]

			internal := make(bson.M)
			cur := typedObj.(bson.M)["bytes currently in the cache"]
			internal["bytes currently in the cache"] = cur
			dirty := typedObj.(bson.M)["bytes dirty in the cache cumulative"]
			internal["bytes dirty in the cache cumulative"] = dirty
			read := typedObj.(bson.M)["bytes read into cache"]
			internal["bytes read into cache"] = read
			write := typedObj.(bson.M)["bytes written from cache"]
			internal["bytes written from cache"] = write

			output["wiredTiger.cache"] = internal
		}
	}

	indentedData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	utils.AppendFile(filepath.Join(dir, db.Name()), "_", indentedData, true)
}
