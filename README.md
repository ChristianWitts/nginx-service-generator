# nginx-service-generator
A simple upstream service generator from zookeeper configuration metadata

### Purpose

To dynamically generate nginx upstream proxy configuration based on your zookeeper service discovery meta-data

### Features

* Checks in with Zookeeper every 10 seconds
* Hashes the configuration so as not to rewrite unless needed
* Can use a configuration file, as well as optionally overriding parameters via the command line

### Releases

* [Version 0.2.0](releases/tag/v0.2.0)
* [Version 0.1.0](releases/tag/v0.1.0)

### Example

Given a zookeeper node information layout as
```
/services/<my-service-name>/instances
[my-services-server_port]
```
and a template file like the default
```
upstream {{.Service}} {
    {{range .UpstreamEndpoints}}server {{.}};{{end}}
}

server {
    listen {{.ListenPort}};
    server_name {{.HostFQDN}};

    location / {
        proxy_set_header HOST               $host;
        proxy_set_header X-Forwarded-Proto  $scheme;
        proxy_set_header X-Real-IP          $remote_addr;
        proxy_set_header X-Forwarded-For    $proxy_add_x_forwarded_for;

        proxy_pass http://{{.Service}};
    }
}
```
a config file that looks like
```
[default]
nginx-root = /etc/nginx
zookeeper-nodes = 127.0.0.1:2181
service-root = /services
service-check-interval = 10
nginx-reload-command = sv reload nginx
fqdn-prefix = api
fqdn-postfix = example.com
listen-port = 80
```
running the following command
```
./service-generator
```
will yield a service like
```
upstream my-service-name {
    server my-services-server:port;
}

server {
    listen 80;
    server_name api.my-service-name.example.com;

    location / {
        proxy_set_header HOST               $host;
        proxy_set_header X-Forwarded-Proto  $scheme;
        proxy_set_header X-Real-IP          $remote_addr;
        proxy_set_header X-Forwarded-For    $proxy_add_x_forwarded_for;

        proxy_pass http://my-service-name;
    }
}
```

If you wanted to override one or more of your configuration file flags, you can optionally specify a command line parameter, such as
```
./service-generator -fqdn-postfix example2.com

# Will yield a service file looking like

upstream my-service-name {
    server my-services-server:port;
}

server {
    listen 80;
    server_name api.my-service-name.example2.com;

    location / {
        proxy_set_header HOST               $host;
        proxy_set_header X-Forwarded-Proto  $scheme;
        proxy_set_header X-Real-IP          $remote_addr;
        proxy_set_header X-Forwarded-For    $proxy_add_x_forwarded_for;

        proxy_pass http://my-service-name;
    }
}
```

The resultant configuration file will be soft-linked from `sites-available` to `sites-enabled`, and nginx will get reloaded so that the configuration is up to date.

This will continue running in the background, at the interval specified by `service-check-interval` in the configuration file, or as a command line parameter.

## TODO

* Add support for removing entire services
