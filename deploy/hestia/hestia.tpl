server {
        listen      %ip%:80;
        server_name %domain_idn% %alias_idn%;
        root        %docroot%;
        access_log  /var/log/%web_system%/domains/%domain%.log combined;
        access_log  /var/log/%web_system%/domains/%domain%.bytes bytes;
        error_log   /var/log/%web_system%/domains/%domain%.error.log error;

        client_max_body_size 12m;
        include %home%/%user%/conf/web/%domain%/nginx.forcessl.conf*;

        location ~ /\.(?!well-known/) {
                deny all;
                return 404;
        }

        location / {
                proxy_pass http://127.0.0.1:3042;
                proxy_http_version 1.1;
                proxy_pass_request_headers on;
                proxy_set_header Host $host;
                proxy_set_header X-Real-IP $remote_addr;
                proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
                proxy_set_header X-Forwarded-Proto $scheme;
                proxy_set_header X-Forwarded-Host $host;
                proxy_set_header X-Forwarded-Port 80;
                proxy_set_header Upgrade $http_upgrade;
                proxy_set_header Connection $connection_upgrade;
                proxy_cache_bypass $http_upgrade;
                proxy_buffering off;
                proxy_read_timeout 180s;
                proxy_send_timeout 180s;
        }

        location /error/ {
                alias %home%/%user%/web/%domain%/document_errors/;
        }

        include %home%/%user%/conf/web/%domain%/nginx.conf_*;
}
