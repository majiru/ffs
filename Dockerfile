FROM golang:1.13.4-alpine3.10
WORKDIR /go/src/github.com/majiru/ffs
ADD . ./

Run mkdir www && echo '<html><body><h2>Hello from simpleblog space</h1></body></html>' > www/index.html

RUN go install github.com/majiru/ffs/cmd/ffs
EXPOSE 8080 4430 5640
CMD /go/bin/ffs 8080 4430 5640 config.json
