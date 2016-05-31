# nginx-service-generator
A simple upstream service generator from zookeeper configuration metadata

### Purpose

To dynamically generate nginx upstream proxy configuration based on your zookeeper service discovery meta-data

### Features

* Checks in with Zookeeper every 10 seconds
* Hashes the configuration so as not to rewrite unless needed

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
    listen 8080;
    server_name api.{{.Service}}.example.com;

    location / {
        proxy_set_header HOST               $host;
        proxy_set_header X-Forwarded-Proto  $scheme;
        proxy_set_header X-Real-IP          $remote_addr;
        proxy_set_header X-Forwarded-For    $proxy_add_x_forwarded_for;

        proxy_pass http://{{.Service}};
    }
}
```
running the following command
```
./generate -zookeeper-nodes 127.0.0.1 -service-root /services -nginx-root /etc/nginx/
```
will yield a configuration like
```
upstream my-service-name {
    server my-services-server:port;
}

server {
    listen 8080;
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

The resultant configuration file will be soft-linked from `sites-available` to `sites-enabled`, and nginx will get reloaded so that the configuration is up to date.

## TODO

* Add support for removing entire services
