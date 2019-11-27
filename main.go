package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

const (
	DataSeparator = "\n"
)

var (
	Host       string
	Port       string
	TimeoutStr string
	Network    string
	Timeout    time.Duration
)

var (
	ForceInterruptConnectionError   = errors.New("close connection intentionally")
	LosingConnectionWithRemoteError = errors.New("connection closed by peer")
)

type TelnetAddr struct {
	network string
	host    string
	port    string
}

func (ta TelnetAddr) Network() string {
	return ta.network
}

func (ta TelnetAddr) String() string {
	return fmt.Sprintf("%s:%s", ta.host, ta.port)
}

type Telnet struct {
	Address        net.Addr
	ConnectTimeout time.Duration
	Context        context.Context
	ContextCancel  context.CancelFunc
	conn           net.Conn
}

func (t *Telnet) open() error {
	ctx, _ := context.WithTimeout(context.Background(), t.ConnectTimeout)
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, t.Address.Network(), t.Address.String())
	if err != nil {
		return err
	}
	t.conn = conn
	return nil
}

func (t *Telnet) close() error {
	if t.conn != nil {
		return t.conn.Close()
	}
	return nil
}

func (t *Telnet) send(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for {
		select {
		case <-t.Context.Done():
			return LosingConnectionWithRemoteError
		default:
			if !scanner.Scan() {
				t.ContextCancel()
				return ForceInterruptConnectionError
			}
			_, err := t.conn.Write([]byte(fmt.Sprintf("%s%s", scanner.Text(), DataSeparator)))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *Telnet) read(w io.Writer) error {
	scanner := bufio.NewScanner(t.conn)
	for {
		select {
		case <-t.Context.Done():
			return ForceInterruptConnectionError
		default:
			if !scanner.Scan() {
				t.ContextCancel()
				return LosingConnectionWithRemoteError
			}
			_, err := w.Write([]byte(fmt.Sprintf("%s%s", scanner.Text(), DataSeparator)))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func newTelnetAddress(network, host, port string) *TelnetAddr {
	return &TelnetAddr{network: network, host: host, port: port}
}

func newTelnet(address TelnetAddr, timeout time.Duration) *Telnet {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	return &Telnet{Address: address, ConnectTimeout: timeout, ContextCancel: cancel, Context: ctx}
}

func init() {
	flag.StringVar(&TimeoutStr, "timeout", "10s", "timeout to connect")
	flag.StringVar(&Network, "network", "tcp", "network type")

	flag.Parse()

	timeout, err := time.ParseDuration(TimeoutStr)
	if err != nil {
		log.Fatalf("invalid timeout %s passed: %s", TimeoutStr, err)
	}

	if len(flag.Args()) < 2 {
		log.Fatal("call format should be like `gotelnet -timeout=10s host port")
	}

	Host = flag.Arg(0)
	Port = flag.Arg(1)
	Timeout = timeout
}

func main() {
	address := newTelnetAddress(Network, Host, Port)
	telnet := newTelnet(*address, Timeout)

	err := telnet.open()
	if err != nil {
		log.Fatal(err)
	}
	defer telnet.close()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := telnet.send(os.Stdin); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := telnet.read(os.Stdout); err != nil {
			log.Fatal(err)
		}
	}()

	wg.Wait()
}
