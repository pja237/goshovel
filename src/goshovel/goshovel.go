package main

import (
	"fmt"
	"net"
    "io"
    "os"
    "sort"
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

// handler for incoming connections
func handleConnection(lconn net.Conn, conf *Config) {
	defer lconn.Close()
	fmt.Println("Got connection from ", lconn.RemoteAddr())

// HERE: call get_next()
    // open connection to backend (R-side) server
    rconn, err:=net.Dial("tcp4","192.168.0.178:22")
    defer rconn.Close()
    if err!=nil {
        fmt.Println(err)
        // HERE: disable backend server
        panic("ERROR: net.Dial()")
    }
    conf.inc_cside()
    conf.inc_sside("server1")
    conf.dump_status()

    // do work l->r
    go shovel(lconn, rconn)
    // do work r->l
    shovel(rconn, lconn)

    conf.dec_cside()
    conf.dec_sside("server1")
    conf.dump_status()
    fmt.Println("EXIT: handleConnection()")
}


// set up data struct to unmarshall from config file
type Server struct {
    Ip string
    Port string
    Enabled bool
    connCount int
}

type Config struct {
    GoShovel Server `yaml:"GoShovel"`
    Backends map[string]*Server `yaml:"Backends"`
}

func (c *Config) dump_status() {
    fmt.Println("--------------------------------------------------------------------------------")
    fmt.Println("Goshovel server (from config):")
    fmt.Println("> ", c.GoShovel.Ip, c.GoShovel.Port, c.GoShovel.Enabled ,c.GoShovel.connCount)
    fmt.Println("Backend servers (from config):")
    for k,v:=range c.Backends {
        fmt.Println("> ",k, " - ", v)
    }
    fmt.Println("--------------------------------------------------------------------------------")
}

func (c *Config) inc_cside() {
    c.GoShovel.connCount++
}

func (c *Config) dec_cside() {
    c.GoShovel.connCount--
}

func (c *Config) inc_sside(s string) {
    c.Backends[s].connCount++
}

func (c *Config) dec_sside(s string) {
    c.Backends[s].connCount--
}

func (c *Config) get_next() {
    //fmt.Println("Len: ", len(c.Backends))
    s:=make([]int, 0, len(c.Backends))
    for _,v:=range c.Backends {
        if v.Enabled==true {
            s=append(s, v.connCount)
        }
    }
    sort.Ints(s)

    for _,v:=range c.Backends {
        if v.Enabled==true && v.connCount==s[0] {
            break
        }
    }
    // open connection to backend (R-side) server

    //fmt.Println("Sorted: ", s)
}

func main() {
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

    // read & unmarshal config
    fi, _:=confFile.Stat()
    config:=make([]byte, fi.Size())
    confFile.Read(config)
    // fmt.Println(config)

    var conf Config
    yaml.Unmarshal(config, &conf)

    conf.dump_status()
    //conf.get_next()

    // starting with the networking stuff
    fmt.Println("Starting listener...")
	//listen, err := net.Listen("tcp4", ":9999")
	fmt.Println("Listening on: ",conf.GoShovel.Ip+":"+conf.GoShovel.Port)
	listen, err := net.Listen("tcp4", conf.GoShovel.Ip+":"+conf.GoShovel.Port)
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
		go handleConnection(lconn, &conf)
	}
}
