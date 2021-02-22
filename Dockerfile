FROM golang:1.15-alpine

WORKDIR /src/app

COPY . .

CMD ["go", "run", "main.go"]
