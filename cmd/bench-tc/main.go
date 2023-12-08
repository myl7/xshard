package main

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"flag"
	"time"

	"github.com/myl7/tcrsa"
	log "github.com/sirupsen/logrus"
)

func main() {
	inShardNum := flag.Int("inShardN", 0, "in-shard node num. must be 3f + 1.")
	tcK := flag.Int("tcK", 0, "threshold of tcrsa. at least half of in-shard node num.")
	flag.Parse()

	if *inShardNum == 0 {
		log.WithField("inShardN", *inShardNum).Fatal("invalid commandline arg")
	}
	if *tcK == 0 {
		log.WithField("tcK", *tcK).Fatal("invalid commandline arg")
	}

	start := time.Now()
	shares, meta, err := tcrsa.NewKey(2048, uint16(*tcK), uint16(*inShardNum), nil)
	if err != nil {
		panic(err)
	}
	t := time.Now()
	elapsed := t.Sub(start)
	log.WithField("elapsed", elapsed).Info("gen")

	sigShares := make([]*tcrsa.SigShare, *inShardNum)
	h := sha256.New()
	h.Write([]byte("hello"))
	b := h.Sum(nil)

	start = time.Now()
	blockBPKCS1, err := tcrsa.PrepareDocumentHash(meta.PublicKey.Size(), crypto.SHA256, b)
	if err != nil {
		panic(err)
	}

	sigShares[0], err = shares[0].Sign(blockBPKCS1, crypto.SHA256, meta)
	if err != nil {
		panic(err)
	}
	t = time.Now()
	elapsed = t.Sub(start)
	log.WithField("elapsed", elapsed).Info("sign")

	for i := 1; i < *inShardNum; i++ {
		sigShares[i], err = shares[i].Sign(blockBPKCS1, crypto.SHA256, meta)
		if err != nil {
			panic(err)
		}
	}

	start = time.Now()
	var sigSharesObj tcrsa.SigShareList = sigShares
	sig, err := sigSharesObj.Join(blockBPKCS1, meta)
	if err != nil {
		panic(err)
	}

	err = rsa.VerifyPKCS1v15(meta.PublicKey, crypto.SHA256, b, sig)
	if err != nil {
		panic(err)
	}
	t = time.Now()
	elapsed = t.Sub(start)
	log.WithField("elapsed", elapsed).Info("verify")
}
