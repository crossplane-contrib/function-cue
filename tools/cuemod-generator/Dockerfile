FROM golang:1.21

WORKDIR /app

COPY generate-cue-mod /app
COPY providers.cue /app

RUN go install cuelang.org/go/cmd/cue@latest && \
    go mod init github.com/crossplane-contrib/function-cue && \
    cue export providers.cue -o providers.json && \
    apt-get update && \
    apt-get install -y python3

CMD ./generate-cue-mod
