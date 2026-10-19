FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/server ./server
COPY static ./static
EXPOSE 8080
CMD ["./server"]
