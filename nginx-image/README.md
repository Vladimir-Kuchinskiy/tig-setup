Build
```bash
docker build -f nginx-image/Dockerfile -t nginx-images ./nginx-image
```

Run
```bash
docker run -p 8080:80 nginx-images
```

Visit a site going to http://localhost:8080/images/image-1.png
