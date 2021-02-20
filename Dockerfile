FROM ubuntu:20.10
RUN apt-get update -y && apt-get upgrade -y && apt-get install libgdal-dev golang-go -y
WORKDIR /app
COPY . .
RUN go build
CMD ["./scraper"]
