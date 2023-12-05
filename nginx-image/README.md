Build
```bash
docker build -t nginx-images .
```

Run
```bash
docker run -p 8080:80 nginx-images
```

Visit a site going to http://localhost:8080/images/image-1.png
