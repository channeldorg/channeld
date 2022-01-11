package channeld

import (
	"flag"
	"strconv"

	"channeld.clewcat.com/channeld/proto"
)

type GlobalSettingsType struct {
	Development bool
	LogLevel    *NullableInt // zapcore.Level
	LogFile     *NullableString

	ServerNetwork string
	ServerAddress string
	ServerFSM     string

	ClientNetwork string
	ClientAddress string
	ClientFSM     string

	CompressionType proto.CompressionType
}

var GlobalSettings = GlobalSettingsType{
	CompressionType: proto.CompressionType_NO_COMPRESSION,
	LogLevel:        &NullableInt{},
	LogFile:         &NullableString{},
}

type NullableInt struct {
	Value    int
	HasValue bool
}

func (i NullableInt) String() string {
	if i.HasValue {
		return strconv.Itoa(i.Value)
	} else {
		return ""
	}
}

func (i *NullableInt) Set(s string) error {
	val, err := strconv.Atoi(s)
	if err == nil {
		i.Value = val
		i.HasValue = true
	}
	return err
}

type NullableString struct {
	Value    string
	HasValue bool
}

func (ns NullableString) String() string {
	return ns.Value
}

func (ns *NullableString) Set(s string) error {
	ns.Value = s
	ns.HasValue = true
	return nil
}

func (s *GlobalSettingsType) ParseFlag() {

	flag.BoolVar(&s.Development, "dev", true, "run in development mode?")
	flag.Var(s.LogLevel, "loglevel", "the log level, -1 = Debug, 0 = Info, 1= Warn, 2 = Error, 3 = Panic")
	//flag.Var(stringPtrFlag{s.LogFile, fmt.Sprintf("logs/%s.log", time.Now().Format("20060102150405"))}, "logfile", "file path to store the log")
	flag.Var(s.LogFile, "logfile", "file path to store the log")

	flag.StringVar(&s.ServerNetwork, "sn", "tcp", "the network type for the server connections")
	flag.StringVar(&s.ServerAddress, "sa", ":11288", "the network address for the server connections")
	flag.StringVar(&s.ServerFSM, "sfsm", "../config/server_authoratative_fsm.json", "the path to the server FSM config")

	flag.StringVar(&s.ClientNetwork, "cn", "tcp", "the network type for the client connections")
	flag.StringVar(&s.ClientAddress, "ca", ":12108", "the network address for the client connections")
	flag.StringVar(&s.ClientFSM, "cfsm", "../config/client_non_authoratative_fsm.json", "the path to the client FSM config")

	ct := flag.Uint("ct", 0, "the compression type, 0 = No, 1 = Snappy")

	flag.Parse()

	if ct != nil {
		s.CompressionType = proto.CompressionType(*ct)
	}
}
