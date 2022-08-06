package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/csv"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/myl7/mingchain/pkg"
)

func main() {
	gCfgPath := flag.String("gcfg-path", "", "Global config path")
	txPath := flag.String("tx-file", "", "Tx file path")
	txRate := flag.Int("tx-rate", 0, "Tx arriven rate")
	flag.Parse()

	if *gCfgPath == "" {
		panic("gcfg-path is required")
	}
	if *txPath == "" {
		panic("tx-file is required")
	}
	if *txRate == 0 {
		panic("tx-rate is required")
	}

	gCfg, err := os.ReadFile(*gCfgPath)
	if err != nil {
		panic(err)
	}

	pkg.LoadConfig(gCfg)

	txFile, err := os.Open(*txPath)
	if err != nil {
		panic(err)
	}
	defer txFile.Close()

	cr := csv.NewReader(txFile)

	fields, err := cr.Read()
	if err != nil {
		panic(err)
	}

	var txes []pkg.Tx
	for {
		row, err := readCsvRow(cr, fields)
		if err != nil {
			if err == io.EOF {
				break
			}

			panic(err)
		}

		blockID, err := strconv.ParseUint(row["blockNumber"], 10, 64)
		if err != nil {
			panic(err)
		}
		fromShard, err := strconv.Atoi(row["fromShard"])
		if err != nil {
			panic(err)
		}
		toShard, err := strconv.Atoi(row["toShard"])
		if err != nil {
			panic(err)
		}

		tx := pkg.Tx{
			BlockID:   blockID,
			TxHash:    row["transactionHash"],
			From:      row["from"],
			FromShard: fromShard,
			To:        row["to"],
			ToShard:   toShard,
			Value:     0,
			IsSubTx:   false,
		}
		txes = append(txes, tx)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}

	go listen()

	i := 0
	for i < len(txes) {
		time.Sleep(1 * time.Second)

		txToSent := make([][]pkg.Tx, pkg.Cfg.ShardNum)

		for j := 0; j < *txRate; j++ {
			tx := txes[i]
			txToSent[tx.ToShard] = append(txToSent[tx.ToShard], tx)

			if tx.FromShard != tx.ToShard {
				subTx := tx
				subTx.IsSubTx = true
				txToSent[subTx.FromShard] = append(txToSent[subTx.FromShard], subTx)
			}

			i++
			if i >= len(txes) {
				break
			}
		}

		log.Println("Sent one round")

		go func() {
			for shard, txes := range txToSent {
				if len(txes) == 0 {
					continue
				}

				msg := pkg.Msg{
					Type:   "request",
					Data:   txes,
					PubKey: key.PublicKey,
				}

				b := pkg.GobEnc(msg)

				pkg.TcpSend(pkg.Cfg.LeaderAddrs[shard], b)
			}
		}()
	}

	log.Println("Done")

	for {
		time.Sleep(60 * time.Second)
		log.Println("1 minute passed")
	}
}

func listen() {
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(err)
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}

		go func(conn net.Conn) {
			defer conn.Close()

			b, err := ioutil.ReadAll(conn)
			if err != nil {
				return
			}

			println("report: " + string(b))
		}(conn)
	}
}

func readCsvRow(cr *csv.Reader, fields []string) (map[string]string, error) {
	row, err := cr.Read()
	if err != nil {
		if err == io.EOF {
			return nil, err
		}

		panic(err)
	}

	obj := make(map[string]string)
	for i, value := range row {
		obj[fields[i]] = value
	}
	return obj, nil
}
