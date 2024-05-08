package object_count

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"k8s.io/klog/v2"
	"kubedb.dev/mongo-doctor/utils"
	"sort"
)

type dbObjCount struct {
	total       int64
	collections []collObjCount
}

type collObjCount struct {
	name  string
	count int64
}

var (
	masterDBObjectCount = make(map[string]dbObjCount)
)

func CompareObjectsCount(pc, so, st *mongo.Client) (int64, error) {
	res := make(map[string]interface{})

	err := pc.Database("admin").RunCommand(context.Background(), bson.D{{Key: "listDatabases", Value: "1"}}).Decode(&res)
	if err != nil {
		return -1, err
	}

	masterDBs, ok := res["databases"]
	if !ok {
		return -1, fmt.Errorf("can't list master databases")
	}

	klog.Info(fmt.Sprintf("Databases listed in the master: %v", len(masterDBs.(primitive.A))))

	for i := 0; i < len(masterDBs.(primitive.A)); i++ {
		masterDB := masterDBs.(primitive.A)[i].(map[string]interface{})
		dbname := masterDB["name"].(string)
		if !utils.SkipDB(dbname) {
			var countDB dbObjCount
			masterDBObjectCount[dbname] = countDB
			countDB.collections = make([]collObjCount, 0)

			collectionNames, err := pc.Database(dbname).ListCollectionNames(context.TODO(), bson.D{})
			if err != nil {
				return -1, err
			}

			for _, collectionName := range collectionNames {
				if !utils.SkipCollection(collectionName) {
					count, err := pc.Database(dbname).Collection(collectionName).EstimatedDocumentCount(context.TODO())
					if err != nil {
						return -1, err
					}
					klog.Info(fmt.Sprintf("master db=%v , collection=%v , docCount=%v", dbname, collectionName, count))

					countDB.total += count
					countDB.collections = append(countDB.collections, collObjCount{name: collectionName, count: count})
				}
			}
			sort.Slice(countDB.collections, func(i, j int) bool {
				return countDB.collections[i].name > countDB.collections[j].name
			})

			masterDBObjectCount[dbname] = countDB
		}
	}

	klog.Info(fmt.Sprintf("Object count done for master "))

	var (
		s1DBObjectCount = make(map[string]dbObjCount)
		s2DBObjectCount = make(map[string]dbObjCount)
	)
	klog.Info(fmt.Sprintf("Object count starting for %s", secondaryOne))
	err = countForSecondary(so, s1DBObjectCount)
	if err != nil {
		return -1, err
	}
	klog.Info(fmt.Sprintf("Object count starting for %s", secondaryTwo))
	err = countForSecondary(st, s2DBObjectCount)
	if err != nil {
		return -1, err
	}
	compare(s1DBObjectCount, s2DBObjectCount)
	return 0, nil
}

func countForSecondary(sc *mongo.Client, sDBMap map[string]dbObjCount) error {
	for dbname, _ := range masterDBObjectCount {
		var countDB dbObjCount
		sDBMap[dbname] = countDB
		countDB.collections = make([]collObjCount, 0)
		collectionNames, err := sc.Database(dbname).ListCollectionNames(context.TODO(), bson.D{})
		if err != nil {
			return err
		}

		for _, collectionName := range collectionNames {
			if !utils.SkipCollection(collectionName) {
				count, err := sc.Database(dbname).Collection(collectionName).EstimatedDocumentCount(context.TODO())
				if err != nil {
					return err
				}

				klog.Info(fmt.Sprintf("currentPod db=%v , collection=%v , docCount=%v", dbname, collectionName, count))
				countDB.total += count
				countDB.collections = append(countDB.collections, collObjCount{name: collectionName, count: count})
			}
		}
		sort.Slice(countDB.collections, func(i, j int) bool {
			return countDB.collections[i].name > countDB.collections[j].name
		})
		sDBMap[dbname] = countDB

		//masterCount := masterDBObjectCount[dbname]
		//replicaCount := objectCounts
		//diff := (math.Abs(float64(masterCount-replicaCount)) / float64(masterCount)) * 100.0
		//klog.Info(fmt.Sprintf("masterCount=%v, replicaCount=%v, diff=%v", masterCount, replicaCount, diff))
		//if diff > 0 {
		//	return replicaCount, fmt.Errorf("object count of database %v didn't match. Master Database Object Count = %v and Replica Database Object count = %v", dbname, masterCount, replicaCount)
		//}
	}
	return nil
}

func compare(m1, m2 map[string]dbObjCount) {
	for dbname, masterDBCount := range masterDBObjectCount {
		m1Count, ok1 := m1[dbname]
		if !ok1 {
			klog.Infof("%s db doesn't exist in pod %s \n", dbname, secondaryOne)
			continue
		}
		m2Count, ok2 := m2[dbname]
		if !ok2 {
			klog.Infof("%s db doesn't exist in pod %s \n", dbname, secondaryTwo)
			continue
		}

		if masterDBCount.total == m1Count.total && masterDBCount.total == m2Count.total {
			klog.Infof("DB %v is fully matched!!! total object count = %v. \n", dbname, masterDBCount.total)
			continue
		}
		klog.Infof("Count didn't matched for DB %v. In %v=%v, %v=%v, %v=%v \n", dbname,
			primaryPod, masterDBCount.total, secondaryOne, m1Count.total, secondaryTwo, m2Count.total)

		klog.Infof("collections for db=%s in %s -> %v \n", dbname, primaryPod, masterDBCount.collections)
		klog.Infof("collections for db=%s in %s -> %v \n", dbname, secondaryOne, m1Count.collections)
		klog.Infof("collections for db=%s in %s -> %v \n", dbname, secondaryTwo, m2Count.collections)
	}
}
