package main

import (
	"github.com/VitZhou/redis-green/proxy"
	"log"
)

func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Print(err)
		}
	}()
	proxy.NewSocketProxy()
	c := make(chan int)
	<-c
}
