package main

import (
	"bytes"
	"crypto/md5"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"text/template"
	"time"

	"github.com/robfig/cron"
	"github.com/samuel/go-zookeeper/zk"
)

//go:generate go run ./scripts/package-templates.go

var (
	templateFile     string
	nginxRoot        string
	zookeeperNodes   string
	serviceRoot      string
	t                *template.Template
	renderedTemplate bytes.Buffer
	sitesAvailable   string
	sitesEnabled     string
	hashes           [string]string
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
}

// updateService will iterate through the services available in zookeeper
// and generate a new template for them.
func updateService(zookeeper *zk.Conn, serviceRoot string) {
	children, _, _, err := zookeeper.ChildrenW(serviceRoot)
	check(err)

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

		fmt.Printf("%+v\n", upstreamEndpoints)

		f, err := os.Create(fmt.Sprintf("%s/%s.service", sitesAvailable, child))
		check(err)
		defer f.Close()

		data := Config{
			Service:           child,
			UpstreamEndpoints: upstreamEndpoints,
		}

		t.Execute(&renderedTemplate, data)
		reload := rewriteConfig(child)
		if reload == true {
			reloadNginx
		}
	}
}

// rewriteConfig will check if the configuration file needs to be overwritten
// and if it's overwritten it needs to signal that nginx must be reloaded
func rewriteConfig(service string) bool {
	renderedHash := md5.Sum([]byte(renderedTemplate.String()))
	if val, ok := hashes[service]; ok {
		if val != renderedHash {
			hashes[service] = renderedHash
		} else {
			return false
		}
	} else {
		hashes[child] = renderedHash
	}
	return true
}

// symlink is a wrapper on os.Symlink
func symlink(service string) {
	err := os.Symlink(fmt.Sprintf("%s/%s.service", sitesAvailable, service),
		fmt.Sprintf("%s/%s.service", sitesEnabled, service))
	check(err)
}

// reloadNginx is a wrapper to shell out to reload the configuration
func reloadNginx() {
	reloadCommand := exec.Command("service", "nginx reload")
	err := reloadCommand.Run()
	check(err)
}

func main() {
	flag.StringVar(&templateFile, "template", "", "nginx template to use")
	flag.StringVar(&nginxRoot, "nginx-root", "/etc/nginx/", "The root of the nginx installation")
	flag.StringVar(&zookeeperNodes, "zookeeper-nodes", "127.0.0.1:2181", "The zookeeper instance to connect to")
	flag.StringVar(&serviceRoot, "service-root", "/", "The root path with your service metadata")
	flag.Parse()

	sitesAvailable = fmt.Sprintf("%s/sites-available/", nginxRoot)
	sitesEnabled = fmt.Sprintf("%s/sites-enabled/", nginxRoot)
	// reloadCommand := exec.Command("service", "nginx reload")

	if templateFile == "" {
		t = template.New(defaultService)
	} else {
		t, err = template.ParseFiles(templateFile)
		check(err)
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	c, _, err := zk.Connect([]string{zookeeperNodes}, time.Second)
	check(err)

	c := cron.New()
	c.AddFunc("*/10 * * * *", func() {
		updateService(c, serviceRoot)
	})
	c.Start()

	defer c.Stop()
	<-sigc
}
