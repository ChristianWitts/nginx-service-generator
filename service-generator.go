package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

//go:generate go run ./scripts/package-templates.go

var (
	templateFile   string
	nginxRoot      string
	zookeeperNodes string
	serviceRoot    string
	t              *template.Template
	err            error
)

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

func main() {
	flag.StringVar(&templateFile, "template", "", "nginx template to use")
	flag.StringVar(&nginxRoot, "nginx-root", "/etc/nginx/", "The root of the nginx installation")
	flag.StringVar(&zookeeperNodes, "zookeeper-nodes", "127.0.0.1:2181", "The zookeeper instance to connect to")
	flag.StringVar(&serviceRoot, "service-root", "/", "The root path with your service metadata")
	flag.Parse()

	sitesAvailable := fmt.Sprintf("%s/sites-available/", nginxRoot)
	sitesEnabled := fmt.Sprintf("%s/sites-enabled/", nginxRoot)
	reloadCommand := exec.Command("service", "nginx reload")

	if templateFile == "" {
		t = template.New(defaultService)
	} else {
		t, err = template.ParseFiles(templateFile)
		check(err)
	}

	c, _, err := zk.Connect([]string{zookeeperNodes}, time.Second)
	check(err)

	children, _, _, err := c.ChildrenW(serviceRoot)
	check(err)

	for _, child := range children {
		serviceInstance := strings.Join([]string{serviceRoot, child, "instances"}, "/")
		info, _, _, err := c.ChildrenW(serviceInstance)
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

		t.Execute(f, data)

		os.Symlink(fmt.Sprintf("%s/%s.service", sitesAvailable, child),
			fmt.Sprintf("%s/%s.service", sitesEnabled, child))
	}

	err = reloadCommand.Run()
	check(err)

}
