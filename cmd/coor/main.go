package main

import "github.com/myl7/xshard"

func main() {
	nd := mingchain.NewCoorNode()
	go nd.SetupNodes()
	nd.Listen()
}
