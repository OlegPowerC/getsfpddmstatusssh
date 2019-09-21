package main

import (
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh"
	"strings"
	"os"
	"bytes"
	"strconv"
	"encoding/xml"
	"sort"
)

type result struct {
	Channel string      `xml:"channel"`
	IsFloat string `xml:"float"`
	Value string `xml:"value"`
	Unit string `xml:"unit"`
	CustomUnit string `xml:"CustomUnit"`
}

type prtgbody struct {
	XMLName xml.Name `xml:"prtg"`
	Error int `xml:"error"`
	TextField string `xml:"text"`
	Res []result `xml:"result"`
}


type Mcmd struct {
	Shtranseivercmdpre string
	Shtranseivercmdseparator string
	Shtranseivercmdpost string
}

type Transeiverdata struct {
	RxLevel float64
	TxLevel float64
	Temperature float64
	Voltage float64
}

var (
	user = flag.String("u", "", "User name")
	host = flag.String("h", "", "Host")
	port = flag.Int("p", 22, "Port")
	passwd = flag.String("pw", "", "Password")
	interfaces = flag.String("i", "", "Interfaces separate by comma")
	debuggmode = flag.Int("d", 0, "Debugg mode: 0 disable, 1 enable")
)

func RetXMLfromMap(IMap map[string]Transeiverdata,buff bytes.Buffer,intnames []string)[]byte{
	var rd1 []result
	StringResult := buff.String()
	RowArBuff := strings.Split(StringResult,"\n")
	for _,rowinres := range RowArBuff{
		sstr := strings.Trim(rowinres," ")
		for _,intname := range intnames{
			if strings.HasPrefix(sstr,intname){
				spstring := strings.Fields(sstr)
				iTemperature,_ := strconv.ParseFloat(spstring[1],64)
				iVoltage,_ := strconv.ParseFloat(spstring[2],64)
				iTXlevel,_ := strconv.ParseFloat(spstring[3],64)
				iRXlevel,_ := strconv.ParseFloat(spstring[4],64)
				IMap[intname] = Transeiverdata{Temperature:iTemperature,Voltage:iVoltage,RxLevel:iRXlevel,TxLevel:iTXlevel}
			}
		}
	}
	keysfsort := make([]string,0,len(IMap))
	for key := range IMap{
		keysfsort = append(keysfsort,key)
	}
	sort.Strings(keysfsort)
	for _,dk := range keysfsort{
		RxLevelString := fmt.Sprintf("%.1f",IMap[dk].RxLevel)
		TxLevelString := fmt.Sprintf("%.1f",IMap[dk].TxLevel)
		rd1 = append(rd1, result{Channel:dk+" RxLevel",IsFloat:"1",Value:RxLevelString,Unit:"Custom",CustomUnit:"dBm"})
		rd1 = append(rd1, result{Channel:dk+" TxLevel",IsFloat:"1",Value:TxLevelString,Unit:"Custom",CustomUnit:"dBm"})
	}
	mt1 := &prtgbody{TextField:"",Res: rd1}
	bolB, _ := xml.Marshal(mt1)
	return bolB
}
func ErrorInProgram (err error){
	var estring string
	estring = fmt.Sprintf("%s",err)
	mt1 := &prtgbody{TextField:estring,Error:1}
	bolB, _ := xml.Marshal(mt1)
	fmt.Println(string(bolB))
	os.Exit(1)
}

func main() {
	var TData map[string]Transeiverdata
	TData = make(map[string]Transeiverdata,0)

	CMcmd := Mcmd{Shtranseivercmdpre:"sh interface",Shtranseivercmdseparator:" ",Shtranseivercmdpost:"transceiver"}
	var Cmds []string
	Cmds = make([]string,0)

	flag.Parse()
	config := &ssh.ClientConfig{
		User: *user,
		Auth: []ssh.AuthMethod{
			ssh.Password(*passwd),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),

	}

	addr := fmt.Sprintf("%s:%d", *host, *port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		ErrorInProgram(err)
	}

	var interfacessplits []string
	if len(*interfaces) > 3{
		interfacessplits = strings.Split(*interfaces,",")
		for _,stint := range interfacessplits{
			Cmds = append(Cmds,CMcmd.Shtranseivercmdpre+CMcmd.Shtranseivercmdseparator+stint+CMcmd.Shtranseivercmdseparator+CMcmd.Shtranseivercmdpost)
		}
	}else{
		os.Exit(1)
	}

	// Create sesssion
	sess, err := client.NewSession()
	if err != nil {
		//fmt.Println("Error")
		ErrorInProgram(err)
	}
	defer sess.Close()

	// StdinPipe for commands
	stdin, err := sess.StdinPipe()
	if err != nil {
		ErrorInProgram(err)
	}

	var somibuff bytes.Buffer
	sess.Stdout = &somibuff
	sess.Stderr = os.Stderr

	// Start remote shell
	err = sess.Shell()
	if err != nil {
		ErrorInProgram(err)
	}

	for _, cmd := range Cmds {
		_, err = fmt.Fprintf(stdin, "%s\n", cmd)
		if err != nil {
			ErrorInProgram(err)
		}
	}

	_, err = fmt.Fprintf(stdin, "%s\n", "exit")

	// Wait for sess to finish
	err = sess.Wait()
	if err != nil {
		ErrorInProgram(err)
	}

	bolB := RetXMLfromMap(TData,somibuff,interfacessplits)
	fmt.Println(string(bolB))
}
