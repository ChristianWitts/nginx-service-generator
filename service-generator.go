package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"text/template"
	"time"

	"github.com/robfig/cron"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/vharitonsky/iniflags"
)

//go:generate go run ./scripts/package-templates.go

type defaultLogger struct{}

func (defaultLogger) Printf(format string, a ...interface{}) {
	log.Printf(format, a...)
}

// Version information
var (
	version    = "0.3.0"
	buildstamp string
	githash    string
)

// Input flags
var (
	templateFile         = flag.String("template", "", "nginx template to use")
	nginxRoot            = flag.String("nginx-root", "/etc/nginx/", "The root of the nginx installation")
	zookeeperNodes       = flag.String("zookeeper-nodes", "127.0.0.1:2181", "The zookeeper instance to connect to")
	serviceRoot          = flag.String("service-root", "/", "The root path with your service metadata")
	serviceCheckInterval = flag.Int("service-check-interval", 10, "The frequency of checking for updated service configuration")
	nginxReloadCommand   = flag.String("nginx-reload-command", "sv reload nginx", "The command that reloads your nginx configuration")
	fqdnPrefix           = flag.String("fqdn-prefix", "api", "The prefix you're using for your Host header")
	fqdnPostfix          = flag.String("fqdn-postfix", "example.com", "The postfix you're using for your Host header")
	listenPort           = flag.Int("listen-port", 80, "The port nginx will listen on")
	printVersion         = flag.Bool("version", false, "Print version information and exit")
)

// Some globally avilable variables
var (
	t                *template.Template
	renderedTemplate bytes.Buffer
	sitesAvailable   string
	sitesEnabled     string
	logger           defaultLogger
)

// check is a simple wrapper to avoid the verbosity of
// `if err != nil` checks.
func check(err error) {
	if err != nil {
		panic(err)
	}
}

// Config is a simple mapping of service name and upstream endpoints
// so that we can generate nginx service configuration
type Config struct {
	Service           string
	UpstreamEndpoints []string
	HostFQDN          string
	ListenPort        int
}

// Service defines the service name, current configuration and template hash,
// as well as if it is currently active which is used to enable/disable the VHost
type Service struct {
	Config Config
	Active bool
	Hash   string
}

var services = make(map[string]Service)

// updateService will iterate through the services available in zookeeper
// and generate a new template for them.
func updateService(zookeeper *zk.Conn, serviceRoot string) {
	children, _, _, err := zookeeper.ChildrenW(serviceRoot)
	check(err)
	reload := false

	for _, child := range children {
		serviceInstance := strings.Join([]string{serviceRoot, child, "instances"}, "/")
		info, _, _, err := zookeeper.ChildrenW(serviceInstance)
		if err != nil {
			panic(err)
		}

		var upstreamEndpoints []string

		for _, instanceInfo := range info {
			i := strings.Join(strings.Split(instanceInfo, "_"), ":")
			upstreamEndpoints = append(upstreamEndpoints, i)
		}

		data := Config{
			Service:           child,
			UpstreamEndpoints: upstreamEndpoints,
			HostFQDN:          strings.Join([]string{*fqdnPrefix, child, *fqdnPostfix}, "."),
			ListenPort:        *listenPort,
		}

		t.Execute(&renderedTemplate, data)
		r := rewriteConfig(child, data)
		if r == true {
			writeOutput(child)
			symlink(child)
			reload = true
		}

		renderedTemplate.Reset()
	}
	if reload == true {
		reloadNginx()
	}
}

// writeOutput will check if the service exists, remove and recreate it
func writeOutput(service string) {
	fname := fmt.Sprintf("%s/%s.service", sitesAvailable, service)
	err := os.Remove(fname)
	f, err := os.OpenFile(fname, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0444)
	check(err)
	_, err = f.WriteString(renderedTemplate.String())
	check(err)
	f.Close()
}

// rewriteConfig will check if the configuration file needs to be overwritten
// and if it's overwritten it needs to signal that nginx must be reloaded
func rewriteConfig(service string, config Config) bool {
	hasher := md5.New()
	hasher.Write([]byte(renderedTemplate.String()))
	renderedHash := hex.EncodeToString(hasher.Sum(nil))

	if val, ok := services[service]; ok {
		if val.Hash == renderedHash {
			logger.Printf("%+v :: %+v unchanged. Not updating.", service, renderedHash)
			return false
		}
	}

	services[service] = Service{
		Config: config,
		Hash:   renderedHash,
		Active: len(config.UpstreamEndpoints) > 0,
	}
	logger.Printf("%+v :: %+v changed. Updating", service, renderedHash)
	return true
}

// symlink is a wrapper on os.Symlink
func symlink(service string) {
	_, err := os.Lstat(fmt.Sprintf("%s/%s.service", sitesEnabled, service))
	if err != nil && services[service].Active {
		// Symlink doesn't already exists, lets create it
		err = os.Symlink(fmt.Sprintf("%s/%s.service", sitesAvailable, service),
			fmt.Sprintf("%s/%s.service", sitesEnabled, service))
		check(err)
	} else {
		// Symlink exists, and the service is no longer active
		os.Remove(fmt.Sprintf("%s/%s.service", sitesEnabled, service))
	}
}

// reloadNginx is a wrapper to shell out to reload the configuration
func reloadNginx() {
	command := strings.Split(*nginxReloadCommand, " ")
	reloadCommand := exec.Command(command[0], command[1:]...)
	err := reloadCommand.Run()
	check(err)
}

func main() {
	iniflags.Parse()
	if *printVersion == true {
		fmt.Printf("service-generator version %s\n", version)
		fmt.Printf("Build datestamp: %+v\n", buildstamp)
		fmt.Printf("Git Hash: %+v\n", githash)
		os.Exit(0)
	}

	sitesAvailable = fmt.Sprintf("%s/sites-available/", *nginxRoot)
	sitesEnabled = fmt.Sprintf("%s/sites-enabled/", *nginxRoot)

	var err error
	if len(*templateFile) == 0 {
		t, err = template.New("service-template").Parse(defaultService)
		check(err)
	} else {
		t, err = template.New("service-template").ParseFiles(*templateFile)
		check(err)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	zookeeper, _, err := zk.Connect([]string{*zookeeperNodes}, time.Second)
	check(err)

	c := cron.New()
	c.AddFunc(fmt.Sprintf("*/%d * * * *", *serviceCheckInterval), func() {
		updateService(zookeeper, *serviceRoot)
	})
	c.Start()

	defer c.Stop()
	<-sigc
}
