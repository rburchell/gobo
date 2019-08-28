package main

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"io/ioutil"
	"log"
)

func main() {
	f, err := ioutil.ReadFile("../stream.out")
	if err != nil {
		panic(fmt.Sprintf("read: %s", err))
	}

	log.Printf("%+v", f)

	var pf PlaybackFile
	err = proto.Unmarshal(f, &pf)
	if err != nil {
		panic(fmt.Sprintf("unmarshal: %s", err))
	}
	log.Printf("PlaybackFile: %+v", pf)
	log.Printf("Body: %s", pf.Body)
}
