package main

import "github.com/myl7/xshard"

func main() {
	nd := xshard.NewCoorNode()
	go nd.SetupNodes()
	nd.Listen()
}
