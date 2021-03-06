package main 

const(
defaultService = `upstream {{.Service}} {
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
`
)
