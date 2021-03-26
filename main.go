package main

import (
	"fmt"
	"github.com/urfave/cli"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

const address = "127.0.0.1"

var (
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

type service struct {
	domainPrefix	string
	port			int
}

type sandbox struct {
	name	string
	dev		bool
}

func sandboxFromContext(c *cli.Context) (*sandbox, error) {
	sandboxName := c.String("sandbox-name")

	if sandboxName == "" {
		return nil, fmt.Errorf("sandbox-name options is missing")
	}

	return &sandbox{
		name:	sandboxName,
		dev:	c.Bool("dev"),
	}, nil
}

func (service *service) listen() error {
	router := service.router()
	return http.ListenAndServe(fmt.Sprintf("%s:%d", address, service.port), router)
}

func initLogger(
	traceHandle io.Writer,
	infoHandle io.Writer,
	warningHandle io.Writer,
	errorHandle io.Writer) {

	Trace = log.New(traceHandle,
		"TRACE: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Info = log.New(infoHandle,
		"INFO: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Warning = log.New(warningHandle,
		"WARNING: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	Error = log.New(errorHandle,
		"ERROR: ",
		log.Ldate|log.Ltime|log.Lshortfile)
}

func startService(service service, sandbox *sandbox, wg *sync.WaitGroup) {
	Info.Printf("Sandbox '%s' is listening on %s:%d for " +
		"'%s.pdok.nl' requests...\n", sandbox.name, address, service.port, service.domainPrefix)

	err := service.listen()
	if err != nil {
		Error.Println(err.Error())
	}

	wg.Done()
}

func main() {
	initLogger(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)

	services := []service{
		{
			domainPrefix: "service",
			port: 6443,
		},
		{
			domainPrefix: "download",
			port: 6444,
		},
		{
			domainPrefix: "api",
			port: 6445,
		},
		{
			domainPrefix: "app",
			port:         6446,
				},
		{
			domainPrefix: "delivery",
			port: 6447,
		},
		{
			domainPrefix: "s3.delivery",
			port: 6448,
		},
	}

	app := cli.NewApp()
	app.Name = "Sandbox Proxy"
	app.Usage = "This Sandbox Proxy is used to setup a local tunnel to the PDOK sandbox environment. " +
		"This proxy handles both routing and security."

	app.HideVersion = true
	app.HideHelp = true

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "sandbox-name",
			Usage:  "Name of the sandbox environment",
			EnvVar: "SANDBOX_NAME",
		},
		cli.BoolFlag{
			Name: "dev",
			Usage: "Set this option to true, to connect to your local development sandbox",
			EnvVar: "DEV",
		},
	}

	app.Action = func(c *cli.Context) error {
		Info.Printf("Starting %s\n", app.Name)

		sandbox, err := sandboxFromContext(c)
		if err != nil {
			cli.ShowAppHelp(c)
			return err
		}

		wg := new(sync.WaitGroup)
		wg.Add(len(services))

		for _, service := range services {
			go startService(service, sandbox, wg)
		}

		wg.Wait()
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		Error.Println(err.Error())
	}
}
