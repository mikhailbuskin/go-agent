docker rm -f test-site

REM Initialize SQLite DB in the mounted folder then start the Apache PHP container
REM Run the init script using a temporary PHP CLI container (writes to ./web/data.db)
docker run --rm -v %cd%/web:/work/web php:8.2-cli php /work/web/init_db.php

docker run -d --name test-site -p 8082:80 -v ./web:/var/www/html php:8.2-apache