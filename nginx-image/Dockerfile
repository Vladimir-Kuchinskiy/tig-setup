FROM emcniece/nginx-cache-purge:1.13-alpine

# Copy custom Nginx configuration file to the container
COPY nginx.conf /etc/nginx/conf.d/default.conf

COPY ./images/ /data/images

# Expose the Nginx HTTP port
EXPOSE 80

# Start Nginx in the foreground
CMD ["nginx", "-g", "daemon off;"]