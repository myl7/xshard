package main

import "github.com/myl7/mingchain"

func main() {
	nd := mingchain.NewCoorNode()
	go nd.SetupNodes()
	nd.Listen()
}
