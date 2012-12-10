package main

import (
	"time"
	"fmt"
	"os"
	"net"
	"strconv"
	"math/rand"
	"math"
)

//In this implementation, we are going to use the Luby Transform code

const numbBlocks = 10.0
const magicNumber = 86


type datagram struct {
	numbOfBlocks uint
	blocks [][]byte
	trailing0s uint
}

type block struct {
	data []byte
	sources []uint8
}


func main() {
	fmt.Println("I compiled!")
	if len(os.Args) != 3{
		fmt.Println("No target file and/or target host argument provided")
		os.Exit(1)
	}
	inFile, _ := os.Open(string(os.Args[1]))
	defer inFile.Close()

	buf := make([]byte, 2048)
	in, _ := inFile.Read(buf)
	dg := encodeFile(buf[0:in])
	fmt.Println("Length: ", dg.Length())
	rand.Seed(int64(22))
	fmt.Println("Hawkward... ",rand.Int31n(int32(10)))
	b1, b2 := byte(4), byte(12)
	fmt.Println("B1", b1, "B2", b2)
	fmt.Println("Well shit ", b1 ^ b2)
	fmt.Println("Test block: ", dg.makeBlock())
	dg.Transmit(os.Args[2])
	for _, val := range(dg.blocks) {
		fmt.Fprintf(os.Stdout, "0x%x\n", val)
	}

}

func (d datagram) Length() (int) {
	return len(d.blocks[uint(numbBlocks-2)])
}

func encodeFile(data []byte) (*datagram) {
	blockSize := uint(math.Ceil(float64(len(data))/numbBlocks))
	if len(data) < numbBlocks {
		fmt.Println("Well this is a bit dumb with that small of an input isn't it?")
		os.Exit(1)
	}
	cont := make([][]byte, numbBlocks)
	for i:=uint(0);i<numbBlocks-1;i++{
		cont[i] = data[(i*blockSize):(i+1)*blockSize]
	}
	tempSlice := data[(numbBlocks-1)*blockSize:]
	trailing0s := blockSize - uint(len(tempSlice))
	cont[int(numbBlocks-1)] = append(tempSlice, make([]byte, trailing0s)...)
	return &datagram{numbBlocks, cont, trailing0s}
}

func getRandomNumber(k uint8) (uint8) { //Gets random number according to the ideal solition distribution
	invValue := rand.Float64()
	val := uint8(math.Ceil(1.0/invValue))
	if val > k{
		val = 1
	}
	return val
}

func (d datagram) makeBlock() (block){
	num := getRandomNumber(uint8(d.numbOfBlocks-1))
	
	cont := make([]byte,len(d.blocks[0]))
	containsSet := make(map[int]struct{})	
	for {
		choice := rand.Intn(int(d.numbOfBlocks))
		if _, ok := containsSet[choice]; !ok {
			cont = xorSlice(cont, d.blocks[choice])
			containsSet[choice] = struct{}{}
		}
		if uint8(len(containsSet)) == num{
			break
		}
	}
	sources := make([]uint8, len(containsSet))
	i := 0
	for k,_ := range(containsSet){
		sources[i] = uint8(k)
		i++
	}
	return block{cont, sources}
}

func xorSlice(b1 []byte, b2 []byte) ([]byte) { 
	if len(b1)!=len(b2){
		panic("Can't compare two blocks of different sizes")
	}
	res := make([]byte, len(b1))
	for i :=range(b1) {
		res[i] = b1[i] ^ b2[i]
	}
	return res
}

func (d datagram) Transmit(dest string) {
	udpAddr, err := net.ResolveUDPAddr("up4", dest)
	checkError(err)

	conn, err := net.DialUDP("udp", nil, udpAddr)
	checkError(err)
	handshake(d, conn)
	quit := make(chan int)
	for i:=0;i<1;i++ {
		go d.sendBlocks(conn, quit)
	}
	conn.Read(make([]byte, 1))
	fmt.Println("Received a success note!")
	quit <- 0

}

func checkError(err error){
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}

func handshake(d datagram, conn *net.UDPConn){
	handshakePacket := make([]byte, 3)
	handshakePacket[0] = byte(magicNumber)
	handshakePacket[1] = byte(1)
	handshakePacket[2] = byte(d.numbOfBlocks)
	fmt.Println("Writing handshake!")
	_, err := conn.Write(handshakePacket)
	fmt.Println("Wrote handshake!")
	checkError(err)
	var buf [8]byte
	_,_, err = conn.ReadFromUDP(buf[0:])
	fmt.Println("Received response!")
	checkError(err)
	if uint8(buf[0])==255 {
		fmt.Println("Looks like handshake went successfully")
	} else {
		fmt.Fprintf(os.Stderr, "The handshake did not succeed, returned %x\n", buf[0:])
	}
	return 
}

func (b block) serialize() ([]byte) {
	headLength := 1 + len(b.sources) 
	ret := make([]byte, headLength)
	ret[0] = byte(len(b.sources))
	conts := ""
	for i, val := range(b.sources) {
		conts = conts +" " + strconv.FormatUint(uint64(val), 10)
		ret[i+1] = byte(val)
	}

	return append(ret, b.data...)
}

func (d datagram) sendBlocks(conn *net.UDPConn, quit chan int){
	for {
		select {
		case <- quit:
			fmt.Println("Stopping sender")
			return
		default:
			sanityCheck :=d.makeBlock().serialize()
			fmt.Fprintf(os.Stderr,"Sending: %v\n",sanityCheck)
			_, err := conn.Write(sanityCheck)
			checkError(err)
			time.Sleep(100 * time.Millisecond)
		}
	}
}