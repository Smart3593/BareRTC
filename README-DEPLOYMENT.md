# BareRTC Deployment Guide for CloudPanel

This guide will help you deploy BareRTC on CloudPanel with the domain `x.tchatenlive.fr`.

## Prerequisites

- CloudPanel server with root access
- Domain `x.tchatenlive.fr` pointing to your server
- Docker and Docker Compose installed on the server
- SSH access to your CloudPanel server

## Step-by-Step Deployment

### Step 1: Prepare Your Server

1. **SSH into your CloudPanel server**

   ```bash
   ssh root@your-server-ip
   ```

2. **Install Docker and Docker Compose** (if not already installed)

   ```bash
   # Install Docker
   curl -fsSL https://get.docker.com -o get-docker.sh
   sh get-docker.sh

   # Install Docker Compose
   curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
   chmod +x /usr/local/bin/docker-compose
   ```

3. **Create application directory**
   ```bash
   mkdir -p /home/cloudpanel/htdocs/x.tchatenlive.fr
   cd /home/cloudpanel/htdocs/x.tchatenlive.fr
   ```

### Step 2: Upload Your Project

1. **Upload the project files to your server**

   ```bash
   # From your local machine, upload the project
   scp -r /path/to/BareRTC/* root@your-server-ip:/home/cloudpanel/htdocs/x.tchatenlive.fr/
   ```

2. **Set proper permissions**
   ```bash
   chown -R cloudpanel:cloudpanel /home/cloudpanel/htdocs/x.tchatenlive.fr
   chmod +x /home/cloudpanel/htdocs/x.tchatenlive.fr/deploy.sh
   ```

### Step 3: Configure CloudPanel

1. **Log into CloudPanel**

   - Open your browser and go to `https://your-server-ip:8443`
   - Login with your CloudPanel credentials

2. **Create a new site**

   - Go to "Sites" → "Add Site"
   - Domain: `x.tchatenlive.fr`
   - PHP Version: None (we're using Docker)
   - Click "Create"

3. **Configure SSL Certificate**
   - Go to your site settings
   - Click "SSL/TLS"
   - Choose "Let's Encrypt" or upload your own certificate
   - Enable "Force HTTPS"

### Step 4: Deploy the Application

1. **SSH into your server and navigate to the project**

   ```bash
   ssh root@your-server-ip
   cd /home/cloudpanel/htdocs/x.tchatenlive.fr
   ```

2. **Run the deployment script**

   ```bash
   ./deploy.sh
   ```

3. **Verify the deployment**
   ```bash
   docker-compose -f docker-compose.prod.yml ps
   ```

### Step 5: Configure Reverse Proxy

1. **In CloudPanel, go to your site settings**

   - Click "Vhost" → "Proxy"
   - Add a new proxy rule:
     - **Source**: `/`
     - **Target**: `http://127.0.0.1:9000`
     - **Type**: `Proxy`

2. **Add WebSocket support**

   - Add another proxy rule:
     - **Source**: `/ws`
     - **Target**: `http://127.0.0.1:9000`
     - **Type**: `Proxy`
     - **Headers**: Add `Upgrade: $http_upgrade` and `Connection: upgrade`

3. **Add API proxy**

   - Add another proxy rule:
     - **Source**: `/api`
     - **Target**: `http://127.0.0.1:9000`
     - **Type**: `Proxy`

4. **Add polling proxy**
   - Add another proxy rule:
     - **Source**: `/poll`
     - **Target**: `http://127.0.0.1:9000`
     - **Type**: `Proxy`

### Step 6: Configure Firewall

1. **Open required ports**

   ```bash
   # Web traffic (handled by CloudPanel)
   ufw allow 80
   ufw allow 443

   # TURN server ports
   ufw allow 3478/tcp
   ufw allow 3478/udp
   ufw allow 5349/tcp
   ufw allow 5349/udp
   ```

### Step 7: Test Your Deployment

1. **Test the application**

   - Open your browser and go to `https://x.tchatenlive.fr`
   - You should see the BareRTC chat interface

2. **Test WebSocket connection**

   - Open browser developer tools
   - Check the Network tab for WebSocket connections
   - Verify that `/ws` connections are established

3. **Test video calling**
   - Try the video calling feature
   - Check if TURN server is working properly

## Configuration Files

### settings.toml

The main configuration file has been updated for your domain:

- `WebsiteURL = 'https://x.tchatenlive.fr'`
- `CORSHosts = ['https://x.tchatenlive.fr', 'https://www.tchatenlive.fr']`
- `UseXForwardedFor = true` (for CloudPanel proxy)

### TURN Server

The TURN server is configured for WebRTC video calling:

- Ports: 3478 (TCP/UDP), 5349 (TCP/UDP)
- Realm: `x.tchatenlive.fr`
- External IP: Automatically detected

## Monitoring and Maintenance

### View Logs

```bash
# Application logs
docker-compose -f docker-compose.prod.yml logs -f barertc

# TURN server logs
docker-compose -f docker-compose.prod.yml logs -f coturn
```

### Update Application

```bash
cd /home/cloudpanel/htdocs/x.tchatenlive.fr
git pull origin main  # if using git
./deploy.sh
```

### Backup Data

```bash
# Backup database and logs
docker-compose -f docker-compose.prod.yml exec barertc tar -czf /tmp/backup.tar.gz /app/data /app/logs
docker cp barertc_prod:/tmp/backup.tar.gz ./backup-$(date +%Y%m%d).tar.gz
```

## Troubleshooting

### Common Issues

1. **Application not accessible**

   - Check if Docker containers are running: `docker-compose -f docker-compose.prod.yml ps`
   - Check application logs: `docker-compose -f docker-compose.prod.yml logs barertc`
   - Verify proxy configuration in CloudPanel

2. **WebSocket connection failed**

   - Check if `/ws` proxy rule is configured correctly
   - Verify that `Upgrade` and `Connection` headers are set
   - Check browser console for WebSocket errors

3. **Video calling not working**

   - Verify TURN server is running: `docker-compose -f docker-compose.prod.yml logs coturn`
   - Check if TURN ports are open in firewall
   - Verify TURN server configuration in `turnserver.conf`

4. **SSL certificate issues**
   - Check SSL configuration in CloudPanel
   - Verify domain DNS settings
   - Check SSL certificate validity

### Performance Optimization

1. **Enable caching**

   - Static assets are already configured with caching headers
   - Consider using CloudPanel's built-in caching features

2. **Monitor resources**

   ```bash
   # Check container resource usage
   docker stats

   # Check disk usage
   df -h
   ```

3. **Scale if needed**
   - For high traffic, consider running multiple instances
   - Use load balancer for horizontal scaling

## Security Considerations

1. **Firewall configuration**

   - Only open necessary ports
   - Use CloudPanel's built-in security features

2. **SSL/TLS**

   - Always use HTTPS in production
   - Keep SSL certificates up to date

3. **Regular updates**
   - Keep Docker images updated
   - Monitor for security patches

## Support

If you encounter issues:

1. Check the logs: `docker-compose -f docker-compose.prod.yml logs`
2. Verify configuration files
3. Test individual components
4. Check CloudPanel documentation

Your BareRTC chat application should now be running successfully at `https://x.tchatenlive.fr`!
