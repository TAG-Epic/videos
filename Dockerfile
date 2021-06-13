FROM golang:1.16
WORKDIR opt/
COPY go.* ./
RUN go mod download
COPY . ./
RUN go get
RUN go build -o out
CMD ["./out"]
