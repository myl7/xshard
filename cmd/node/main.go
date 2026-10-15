package main

import "github.com/myl7/xshard"

func main() {
	nd := mingchain.NewNode()
	go nd.SendReady()
	go nd.PackBlock()
	nd.Listen()
}
