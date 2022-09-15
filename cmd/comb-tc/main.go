package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/myl7/tcrsa"
	log "github.com/sirupsen/logrus"
)

func main() {
	tcConfigsS := flag.String("tcConfigs", "", "tc config files to be combined")
	flag.Parse()

	if *tcConfigsS == "" {
		log.WithField("tcConfigs", *tcConfigsS).Fatal("invalid commandline arg")
	}

	var tcAllKeys []TcKeys
	tcConfigs := strings.Split(*tcConfigsS, ",")
	for _, tcConfig := range tcConfigs {
		var tcKeys []TcKeys

		// tcrsa cache
		func() {
			f, err := os.Open(tcConfig)
			if err != nil {
				return
			}
			defer f.Close()

			err = gob.NewDecoder(f).Decode(&tcKeys)
			if err != nil {
				panic(err)
			}
		}()

		tcAllKeys = append(tcAllKeys, tcKeys...)
	}

	func() {
		f, err := os.Create(fmt.Sprintf("tc%d-%dx%d", tcAllKeys[0].L, tcAllKeys[0].K, len(tcAllKeys)))
		if err != nil {
			panic(err)
		}
		defer f.Close()

		err = gob.NewEncoder(f).Encode(tcAllKeys)
		if err != nil {
			panic(err)
		}
	}()
}

type TcKeys struct {
	K      int
	L      int
	Meta   *tcrsa.KeyMeta
	Shares []*tcrsa.KeyShare
}
