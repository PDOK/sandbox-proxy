package main

import (
	"crypto/rsa"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/urfave/cli"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const address = "127.0.0.1"

const (
	processing cluster = iota
	services
	monitoring
)

var (
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

type cluster int

type service struct {
	domain 	string
	port   	int
	cluster cluster
}

type sandbox struct {
	name       	string
	bearerToken string
	dev        	bool
}

func (c cluster) String() string {
	return [...]string{"processing", "services", "monitoring"}[c]
}

func sandboxFromContext(c *cli.Context) (*sandbox, error) {
	sandboxName := c.String("sandbox-name")
	privateKey := c.String("private-key")

	if sandboxName == "" {
		return nil, fmt.Errorf("sandbox-name options is missing")
	}

	if privateKey == "" {
		return nil, fmt.Errorf("private-key options is missing")
	}

	signBytes, err := ioutil.ReadFile(privateKey)
	if err != nil {
		return nil, err
	}

	signKey, err := jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	if err != nil {
		return nil, err
	}

	bearerToken, err := generateBearerToken(sandboxName, signKey)
	if err != nil {
		return nil, err
	}

	return &sandbox{
		name:       	sandboxName,
		bearerToken: 	bearerToken,
		dev:        	c.Bool("dev"),
	}, nil
}

func (service *service) listen(sandbox *sandbox, bindAddress string) error {
	router := service.router(sandbox)
	return http.ListenAndServe(fmt.Sprintf("%s:%d", bindAddress, service.port), router)
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

func startService(service service, sandbox *sandbox, wg *sync.WaitGroup, bindAddress string) {
	Info.Printf("Sandbox '%s' is listening on %s:%d for "+
		"'%s' requests...\n", sandbox.name, bindAddress, service.port, service.domain)

	err := service.listen(sandbox, bindAddress)
	if err != nil {
		Error.Println(err.Error())
	}

	wg.Done()
}

func generateBearerToken(iss string, privateKey *rsa.PrivateKey) (string, error) {
	Info.Println("Generating bearer token")

	t := jwt.New(jwt.GetSigningMethod("RS256"))
	t.Claims = &jwt.StandardClaims{
		ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
		Issuer:    iss,
	}

	return t.SignedString(privateKey)
}

func main() {
	initLogger(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)

	services := []service{
		{
			domain:		"service.pdok.nl",
			port:   	5000,
			cluster: 	services,
		},
		{
			domain: 	"download.pdok.nl",
			port:   	5001,
			cluster: 	services,
		},
		{
			domain: 	"api.pdok.nl",
			port:   	5002,
			cluster: 	services,
		},
		{
			domain: 	"app.pdok.nl",
			port:   	5003,
			cluster: 	services,
		},
		{
			domain: 	"delivery.pdok.nl",
			port:   	5004,
			cluster: 	processing,
		},
		{
			domain: 	"s3.delivery.pdok.nl",
			port:   	5005,
			cluster: 	processing,
		},
		{
			domain: 	"pdok.cloud.kadaster.nl",
			port:   	5006,
			cluster: 	monitoring,
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
		cli.StringFlag{
			Name:   "private-key",
			Usage:  "Reference to the private key file used to generate the public key",
			EnvVar: "PRIVATE_KEY",
		},
		cli.BoolFlag{
			Name:   "dev",
			Usage:  "Set this option to true, to connect to your local development sandbox",
			EnvVar: "DEV",
		},
		cli.StringFlag{
			Name:	"bind-address",
			Usage:	"Bind address (default 127.0.0.1)",
			EnvVar: "BIND_ADDRESS",
			Value: 	address,
		},
	}

	app.Action = func(c *cli.Context) error {
		Info.Printf("Starting %s\n", app.Name)

		sandbox, err := sandboxFromContext(c)
		if err != nil {
			cli.ShowAppHelp(c)
			return err
		}

		bindAddress := c.String("bind-address")

		wg := new(sync.WaitGroup)
		wg.Add(len(services))

		for _, service := range services {
			go startService(service, sandbox, wg, bindAddress)
		}

		wg.Wait()
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		Error.Println(err.Error())
	}
}
