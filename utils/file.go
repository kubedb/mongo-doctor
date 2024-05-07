package utils

import (
	"k8s.io/klog/v2"
	"log"
	"os"
)

func MakeDir(dir string) {
	_, err := os.Stat(dir)
	if err == nil {
		klog.Infof("Directory %s already exists; cleaning up.\n", dir)
		err = os.RemoveAll(dir)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		log.Fatal(err)
	}
}

func WriteFile(dir, fileName string, data []byte) {
	fileName = dir + "/" + fileName + ".json"
	err := os.WriteFile(fileName, data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func AppendFile(dir, fileName string, data []byte, comma bool) {
	fileName = dir + "/" + fileName + ".json"
	file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = file.WriteString(",\n")
	if err != nil {
		log.Fatal(err)
	}

	// Write data to the file
	_, err = file.WriteString(string(data))
	if err != nil {
		log.Fatal(err)
	}
}
