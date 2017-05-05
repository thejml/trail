FROM golang

COPY ./ /app/src/

ENV GOPATH=/app/src/
ENV GOBIN=/app/bin/

WORKDIR /app/

# Install requirements
RUN go get gopkg.in/mgo.v2 goji.io

# Compile
RUN go install src/trail.go

# Run
CMD ["bin/trail"]
