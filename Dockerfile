FROM golang:1.19-alpine

WORKDIR $GOPATH/github.com/panagiotisptr/set-echo

COPY . .

WORKDIR $GOPATH/github.com/panagiotisptr/set-echo

RUN go mod download

RUN cd cmd/set-echo && go install

CMD ["set-echo"]
