package nginx

var nginxConfTemplate = `# managed by interlock
user  {{ .User }};
worker_processes  {{ .MaxProcesses }};
worker_rlimit_nofile {{ .RLimitNoFile }};

error_log  /var/log/error.log warn;
pid        {{ .PidPath }};


events {
    worker_connections  {{ .MaxConnections }};
    multi_accept on;
}


http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;
    server_names_hash_bucket_size 64;
    client_max_body_size 2048M;
    types_hash_max_size 2048;

    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    access_log  /var/log/nginx/access.log  main;

    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;

    keepalive_timeout  10s;

    # If we receive X-Forwarded-Proto, pass it through; otherwise, pass along the
    # scheme used to connect to this server
    map $http_x_forwarded_proto $proxy_x_forwarded_proto {
      default $http_x_forwarded_proto;
      ''      $scheme;
    }

    gzip on;
    gzip__vary on;
    gzip_proxied any;
    gzip_comp_level 9;
    gzip_buffers 16 8k;
    gzip_http_version 1.1;
    gzip_types text/plain text/css application/json
    applicationx-javascript text/xml application/xml
    application/xml+rss text/javascript;

    # ssl
    ssl_ciphers {{ .SSLCiphers }};
    ssl_protocols {{ .SSLProtocols }};

    map $http_upgrade $connection_upgrade {
        default upgrade;
        ''      close;
    }

    # default host return 503
    server {
            listen {{ .Port }};
            {{ if ne .Port 8080 }}
            listen 8080;
            {{ end }}
            server_name _;

            location / {
                return 503;
            }

            location /nginx_status {
                stub_status on;
                access_log off;
            }
    }

    {{ range $host := .Hosts }}
    upstream {{ $host.Upstream.Name }} {
        {{ range $up := $host.Upstream.Servers }}server {{ $up.Addr }};
        {{ end }}
    }
    server {
        listen {{ $host.Port }};
        {{ if ne $host.Port 8080 }}
        listen 8080;
        {{ end }}

        server_name{{ range $name := $host.ServerNames }} {{ $name }}{{ end }};
        {{ if $host.SSLOnly }}return 302 https://$server_name$request_uri;{{ else }}
        location / {
            {{ if $host.SSLBackend }}proxy_pass https://{{ $host.Upstream.Name }};{{ else }}proxy_pass http://{{ $host.Upstream.Name }};{{ end }}
        }

        location ~* .(ico|jpg|webp|jpeg|gif|css|png|js|ico|bmp|zip|woff)$ {
            access_log off;
            log_not_found off;
            add_header Pragma public;
            add_header Cache-Control "public";
            expires 14d;
        }

        location ~* .(php|html)$ {
            access_log off;
            log_not_found off;
            add_header Pragma public;
            add_header Cache-Control "public";
            expires 14d;
        }

        {{ range $ws := $host.WebsocketEndpoints }}
        location {{ $ws }} {
            fastcgi_pass_header Set-Cookie;
            fastcgi_pass_header Cookie;
            fastcgi_ignore_headers Cache-Control Expires Set-Cookie;
            fastcgi_index index.php;
            fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
            fastcgi_split_path_info ^(.+.php)(/.+)$;
            fastcgi_param  PATH_INFO $fastcgi_path_info;
            fastcgi_param  PATH_TRANSLATED    $document_root$fastcgi_path_info;
            fastcgi_intercept_errors on;
            include fastcgi_params;

            fastcgi_pass {{ $host.Upstream.Name }};
        }

        location /nginx_status {
            stub_status on;
            access_log off;
        }
        {{ end }}
        {{ end }}
    }
    {{ if $host.SSL }}
    server {
        listen {{ .SSLPort }};
        {{ if ne .SSLPort 8443 }}
        listen 8443;
        {{ end }}
        ssl on;
        ssl_certificate {{ $host.SSLCert }};
        ssl_certificate_key {{ $host.SSLCertKey }};
        server_name{{ range $name := $host.ServerNames }} {{ $name }}{{ end }};

        location / {
            {{ if $host.SSLBackend }}proxy_pass https://{{ $host.Upstream.Name }};{{ else }}proxy_pass http://{{ $host.Upstream.Name }};{{ end }}
        }

        {{ range $ws := $host.WebsocketEndpoints }}
        location {{ $ws }} {
            {{ if $host.SSLBackend }}proxy_pass https://{{ $host.Upstream.Name }};{{ else }}proxy_pass http://{{ $host.Upstream.Name }};{{ end }}
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection $connection_upgrade;
        }

        location /nginx_status {
            stub_status on;
            access_log off;
        }
        {{ end }}
    }
    {{ end }}
    {{ end }}

    include /etc/nginx/conf.d/*.conf;
}
`
