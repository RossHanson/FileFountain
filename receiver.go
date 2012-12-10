package main

import (
	"net"
	"os"
	"fmt"
	"errors"
	"container/list"
)

var packetCount = 0

type partialData struct {
	decodedBlocks [][]byte
	dependencies []*list.List

}

type block struct {
	data []byte 
	sources *list.List
	solved bool
}

func checkError(err error){
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func main()  {
	service := ":1200"
	udpAddr, err := net.ResolveUDPAddr("up4", service)
	checkError(err)

	if len(os.Args)!=2{
		fmt.Fprintf(os.Stdout, "Usage: %v outputFile", os.Args[0])
		return
	}

	listener, err := net.ListenUDP("udp", udpAddr)
	checkError(err)
	p,_:=waitForHandshake(listener)


	for {
		buf := make([]byte,1500)
		fmt.Println("Bout to wait on some reading")
		n, _, err := listener.ReadFromUDP(buf)
		if err != nil {
			continue
		}
		fmt.Println("Spawning a new handler")
		done := p.handleClient(buf[0:n])
		if done {
			finish(p, os.Args[1])
			break
		}
	}
}

func finish(p partialData, filename string){
	fmt.Fprintf(os.Stdout,"Finished! Received %v packets\n",packetCount)
	fi, err := os.Create("out.txt")
	checkError(err)
	defer fi.Close()
	for _,val := range(p.decodedBlocks){
		fmt.Fprintf(os.Stdout,"0x%x\n",val)
		fi.Write(val)
	}	
}

func waitForHandshake(conn *net.UDPConn) (partialData, error){
	var buf [8]byte
	for {
		n, addr, err := conn.ReadFromUDP(buf[0:])
		checkError(err)
		if buf[0] == byte(86){
			fmt.Println("Writing handshake")
			numSources := int(uint(buf[2]))
			fmt.Println("Num of sources: ",numSources)
			dependencies:= make([]*list.List, numSources)
			for i:=0;i<numSources;i++{
				dependencies[i] = list.New()
			}
			decodedBlocks := make([][]byte, numSources)
			conn.WriteToUDP(make([]byte,1),addr)
			fmt.Println("Wrote handshake")
			return partialData{decodedBlocks, dependencies}, nil

			
		} else {
			fmt.Fprintf(os.Stdout,"Received non handshake packet:%p\n", buf[0:n])
		}	
	}
	return partialData{}, errors.New("No handshake received!")

	

}

func (p partialData) handleClient(message []byte) (bool){
	packetCount++
	fmt.Fprintf(os.Stderr,"Received: %v\n", message)
	numSources := int(uint(message[0]))
	sources := list.New()
	for i := 0; i<numSources;i++ {
		sources.PushFront(uint(message[i+1]))
	}
	fmt.Fprintf(os.Stderr,"Message data: %v      Block data: %v",message, message[(numSources+1):])
	b := block{message[(numSources+1):],sources,false}
	solved, num := b.addDependencies(p)
	fmt.Println("Done!")
	if solved {
		p.handleSolved(b, num)
	}
	return p.checkDone()
	
}

func (p partialData) determineRemaing(sources *list.List) (*list.List) {
	remaining := list.New()

	for e := sources.Front(); e != nil; e = e.Next() {
		sourceNum := int(uint(e.Value.(uint)))
		fmt.Println("Source num: ", sourceNum)
		if len(p.decodedBlocks[sourceNum]) == 0 {
			remaining.PushFront(sourceNum)
		}
	}
	fmt.Println("Ret")
	return remaining
}

func (p partialData) handleSolved(b block, sNum int){
	fmt.Println("Handling solved")
	for e:= b.sources.Front();e!=nil;e=e.Next(){
		b.data = xorSlice(b.data, p.decodedBlocks[int(e.Value.(uint))])
	}
	p.decodedBlocks[sNum] = b.data
	for e:=p.dependencies[sNum].Front();e!=nil;e=e.Next(){
		dat :=block(e.Value.(block))
		rem := p.determineRemaing(dat.sources)
		if rem.Len() == 1{
			fmt.Println("handling the solved in handle solved")
			p.handleSolved(dat,int(rem.Front().Value.(int)))
		}
	}
}


func (p partialData) checkDone() (bool) {
	numLeft:=0
	for i, val := range(p.decodedBlocks){
		if len(val)==0{
			fmt.Println("Well shit,",i,"is left")
			numLeft++
		}
	}
	fmt.Println(numLeft, "blocks left")
	return numLeft==0
}

func (b block) addDependencies(p partialData) (bool, int){
	rem := p.determineRemaing(b.sources)

	if rem.Len() == 1{
		return true, int(rem.Front().Value.(int))
	}
	fmt.Println("Bout to add some deps")
	for e:=rem.Front();e!=nil;e=e.Next() {
		p.dependencies[int(e.Value.(int))].PushFront(b)
	}
	fmt.Println("ret")
	return false, -1
}

func xorSlice(b1 []byte, b2 []byte) ([]byte) { 
	if len(b1)==0 && len(b2)!=0{
		return b2
	}
	if len(b1)!=0 && len(b2)==0{
		return b1
	}
	if len(b1)!=len(b2){
		fmt.Fprintf(os.Stderr,"B1 size %v, B2 size %v", len(b1), len(b2))
		panic("Can't compare two blocks of different sizes")
	}
	res := make([]byte, len(b1))
	for i :=range(b1) {
		res[i] = b1[i] ^ b2[i]
	}
	return res
}