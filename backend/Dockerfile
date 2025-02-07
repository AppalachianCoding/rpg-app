FROM golang:1.23-alpine AS builder
WORKDIR /app

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ .
RUN go build -o rpg-app .

FROM alpine:latest
WORKDIR /root/

COPY --from=builder /app/rpg-app .
COPY --from=builder /app/5e_data ./5e_data
RUN chmod +x rpg-app
EXPOSE 80
CMD ["./rpg-app"]
