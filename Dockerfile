FROM golang:1.23-alpine AS base

RUN apk add make bash git nodejs npm

WORKDIR /app

COPY go.mod go.sum package.json package-lock.json ./
RUN go mod download
RUN npm install

COPY . ./

RUN npm run build
RUN make build

CMD ["./BareRTC"]