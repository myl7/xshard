package main

import "github.com/myl7/xshard"

func main() {
	nd := xshard.NewNode()
	go nd.SendReady()
	go nd.PackBlock()
	nd.Listen()
}
