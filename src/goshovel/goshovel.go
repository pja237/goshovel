package main

import (
	"fmt"
	"net"
    "io"
    "os"
    "goshovel/config"
    "gopkg.in/yaml.v2"
)

var ENV_CONFIG string = "GOSHOVEL_CONFIG"
var CONFIG string = "./goshovel.conf"


// shovel data from srcconn --to--> dstconn
func shovel(srcconn, dstconn net.Conn) {
    fmt.Println("Shoveling from: ", srcconn.RemoteAddr(), " --to--> ", dstconn.RemoteAddr())
    _,err:=io.Copy(dstconn, srcconn)
    if err!=nil {
        fmt.Println("ERROR: copy(): ", err)
    } else {
        fmt.Println("Clean Exit from Shovel()")
    }
}

func handleConnection(lconn net.Conn) {
	defer lconn.Close()
	fmt.Println("Got connection from ", lconn.RemoteAddr())

    // open connection to backend (R-side) server
    rconn, err:=net.Dial("tcp4","192.168.0.178:22")
    defer rconn.Close()
    if err!=nil {
        fmt.Println(err)
        panic("ERROR: net.Dial()")
    }

    go shovel(lconn, rconn)
    shovel(rconn, lconn)

    fmt.Println("EXIT: handleConnection()")
}

func main() {
    config.Hello()


    // check env variable for the config file path
    val, present := os.LookupEnv(ENV_CONFIG)
    if !present {
        fmt.Println("ENV[\"GOSHOVEL_CONFIG\"] not present, going with default: ", CONFIG)
    } else {
        CONFIG=val
        fmt.Println("ENV[\"GOSHOVEL_CONFIG\"] == ", CONFIG)
    }

    // open config file
    confFile, err:=os.Open(CONFIG)
    if err!=nil {
        fmt.Println(err)
        panic("ERROR: os.Open()")
    }

    // set up data struct to unmarshall from config file
    type Serv struct {
        Ip string
        Port string
    }

    var srv map[string]Serv

    // read & unmarshal config
    fi, _:=confFile.Stat()
    config:=make([]byte, fi.Size())
    confFile.Read(config)
    fmt.Println(config)

    yaml.Unmarshal(config, &srv)

    for k,v:=range srv {
        fmt.Println(k," - ",v)
    }

    // starting with the networking stuff
    fmt.Println("Starting listener...")
	//listen, err := net.Listen("tcp4", ":9999")
	fmt.Println("Listening on: ",srv["goshovel"].Ip+":"+srv["goshovel"].Port)
	listen, err := net.Listen("tcp4", srv["goshovel"].Ip+":"+srv["goshovel"].Port)
	if err != nil {
        fmt.Println(err)
		panic("ERROR: listen()")
	}
	for {
        // Left Side connection
        // [cli] <--> [goshovel] <--> [ssh server]
        //         L           R
		lconn, err := listen.Accept()
		if err != nil {
            fmt.Println(err)
			panic("ERROR: listen.Accept()")
		}
		go handleConnection(lconn)
	}
}
