# services-demo-basel-2018
Example application used for `One App, Two Platforms, Three Cloud Services` talk on CF Summit Basel 2018.


### Architecture
![](./assets/arch.png)


### Building docker images
```bash
# from the root of the repo
docker build -t servicesapi/web-app:latest . -f web-app/Dockerfile
docker build -t servicesapi/worker-app:latest . -f worker-app/Dockerfile
```
