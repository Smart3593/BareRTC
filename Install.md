# Installing BareRTC

This document will explain how to download and install BareRTC on your own web server.

At this time, BareRTC is not released as a versioned pre-built archive, but as source code. This may change in the future but for now you'll need to git clone or download the source code and compile it, all of which should be easy to do on a Linux or macOS server.

- [Installing BareRTC](#installing-barertc)
  - [Docker Compose](#docker-compose)
  - [Requirements \& Dependencies](#requirements--dependencies)
  - [Installation](#installation)
  - [Deploying to Production](#deploying-to-production)
  - [Developing This App](#developing-this-app)
- [License](#license)

## Docker Compose

There is an easy docker-compose.yml in the git repo:

```bash
docker-compose up
```

Look inside the file for more information.

## Requirements & Dependencies

To run BareRTC on your own website, you will generally need:

* A dedicated server or <abbr title="Virtual Private Server">VPS</abbr> for your web hosting, e.g. with root access to the console to be able to install and configure software.
    * Any Linux distribution or a macOS server will work. You may be able to use a Windows server but this is out of scope for this document and you're on your own there.
    * The BareRTC server is written in pure Go so any platform that the Go language can compile for should work.
    * Note: if you don't have access to manage your server (e.g. you are on a shared web hosting plan with only FTP upload access), you **will not** be able to run BareRTC.
* Recommended: a reverse proxy server like NGINX.

Your server may need programming languages for Go and JavaScript (node.js) in order to compile BareRTC and build its front-end javascript app.

```bash
# Debian or Ubuntu
sudo apt update
sudo apt install golang nodejs npm

# Fedora
sudo dnf install golang nodejs npm

# Mac OS (with homebrew, https://brew.sh)
brew install golang nodejs npm
```

## Installation

The recommended method is to use **git** to download a clone of the source code repository. This way you can update the app by running a `git pull` command to get the latest source.

```bash
# Clone the git repository and change
git clone https://git.kirsle.net/apps/BareRTC
cd BareRTC/

# Compile the front-end javascript single page app
npm install
npm run build

# Compile the back-end Go app to ./BareRTC
make build

# Or immediately run the app from Go source code now
# Listens on http://localhost:9000
make run

# Command line interface to run the binary:
./BareRTC -address :9000 -debug
```

You can also download the repository as a ZIP file or tarball, though updating the code for future versions of BareRTC is a more manual process then.

* ZIP download: https://git.kirsle.net/apps/BareRTC/archive/master.zip
* Tarball: https://git.kirsle.net/apps/BareRTC/archive/master.tar.gz


## Deploying to Production

It is recommended to use a reverse proxy such as nginx in front of this app. You will need to configure nginx to forward WebSocket related headers:

```nginx
server {
    server_name chat.example.com;
    listen 443 ssl http2;
    listen [::]:443 ssl http2;

    ssl_certificate /etc/letsencrypt/live/chat.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/chat.example.com/privkey.pem;

    # Proxy pass to BareRTC.
    location / {
        proxy_pass http://127.0.0.1:9000;

        # WebSocket headers to forward along.
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
    }
}
```

You can run the BareRTC app itself using any service supervisor you like. I use [Supervisor](http://supervisord.org/introduction.html) and you can configure BareRTC like so:

```ini
# /etc/supervisor/conf.d/barertc.conf
[program:barertc]
command = /home/user/git/BareRTC/BareRTC -address 127.0.0.1:9000
directory = /home/user/git/BareRTC
user = user
```

Then `sudo supervisorctl reread && sudo supervisorctl add barertc` to start the app.

## Developing This App

In local development you'll probably run two processes in your terminal: one to `npm run watch` the Vue.js app and the other to run the Go server.

Building and running the front-end app:

```bash
# Install dependencies
npm install

# Build the front-end
npm run build

# Run the front-end in watch mode for local dev
npm run watch
```

And `make run` to run the Go server.

# License

GPLv3.
