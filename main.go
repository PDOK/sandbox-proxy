package main

import (
	"crypto/rsa"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/urfave/cli"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

const address = "127.0.0.1"
const remoteUrl = "https://sandbox.test.pdok.nl"
const defaultAuthorizationHeader = "X-PDOK-Authorization"

const (
	processing Cluster = iota
	services
	monitoring
)

var (
	Trace   *log.Logger
	Info    *log.Logger
	Warning *log.Logger
	Error   *log.Logger
)

type Cluster int

type Service struct {
	domain  string
	port    int
	cluster Cluster
}

type Sandbox struct {
	name        string
	bearerToken string
	remoteUrl   *url.URL
	authHeader  string
}

func (c Cluster) String() string {
	return [...]string{"processing", "services", "monitoring"}[c]
}

func sandboxFromContext(c *cli.Context) (*Sandbox, error) {
	sandboxName := c.String("sandbox-name")
	privateKey := c.String("private-key")
	authHeader := c.String("authorization-header")

	if sandboxName == "" {
		return nil, fmt.Errorf("sandbox-name options is missing")
	}

	if privateKey == "" {
		return nil, fmt.Errorf("private-key options is missing")
	}

	if authHeader == "" {
		authHeader = defaultAuthorizationHeader
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

	remoteUrl, err := url.Parse(c.String("remote-url"))
	if err != nil {
		return nil, err
	}

	return &Sandbox{
		name:        sandboxName,
		bearerToken: bearerToken,
		remoteUrl:   remoteUrl,
		authHeader:  authHeader,
	}, nil
}

func (service *Service) listen(sandbox *Sandbox, bindAddress string) error {
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

func startService(service Service, sandbox *Sandbox, wg *sync.WaitGroup, bindAddress string) {
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
	t.Claims = &jwt.RegisteredClaims{
		ExpiresAt: &jwt.NumericDate{Time: time.Now().Add(time.Hour * 24)},
		Issuer:    iss,
	}

	return t.SignedString(privateKey)
}

func main() {
	initLogger(ioutil.Discard, os.Stdout, os.Stdout, os.Stderr)

	services := []Service{
		{
			domain:  "service.pdok.nl",
			port:    5000,
			cluster: services,
		},
		{
			domain:  "download.pdok.nl",
			port:    5001,
			cluster: services,
		},
		{
			domain:  "api.pdok.nl",
			port:    5002,
			cluster: services,
		},
		{
			domain:  "app.pdok.nl",
			port:    5003,
			cluster: services,
		},
		{
			domain:  "delivery.pdok.nl",
			port:    5004,
			cluster: processing,
		},
		{
			domain:  "s3.delivery.pdok.nl",
			port:    5005,
			cluster: processing,
		},
		{
			domain:  "pdok.cloud.kadaster.nl",
			port:    5006,
			cluster: monitoring,
		},
	}

	app := cli.NewApp()
	app.Name = "Sandbox Proxy"
	app.Usage = "This Sandbox Proxy is used to setup a local tunnel to the PDOK sandbox environment. " +
		"This proxy handles both routing and security."
	app.Version = "0.2"

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
		cli.StringFlag{
			Name:   "remote-url",
			Usage:  fmt.Sprintf("Remote url (default %s)", remoteUrl),
			EnvVar: "REMOTE_URL",
			Value:  remoteUrl,
		},
		cli.StringFlag{
			Name:   "bind-address",
			Usage:  fmt.Sprintf("Bind address (default %s)", address),
			EnvVar: "BIND_ADDRESS",
			Value:  address,
		},
	}

	app.Action = func(c *cli.Context) error {
		sandbox, err := sandboxFromContext(c)
		if err != nil {
			cli.ShowAppHelp(c)
			return err
		}

		Info.Printf("Starting %s\n", app.Name)

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
