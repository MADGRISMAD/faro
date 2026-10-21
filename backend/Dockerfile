FROM golang:1.24-bookworm AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /farocontenido .
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /farocontenido /farocontenido
EXPOSE 18081
ENTRYPOINT ["/farocontenido"]
