FROM golang:1.23-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/novel-studio ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/novel-studio /app/novel-studio
COPY frontend /app/frontend
ENV PORT=9008
EXPOSE 9008
CMD ["/app/novel-studio"]
