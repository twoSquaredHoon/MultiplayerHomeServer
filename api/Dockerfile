FROM golang:1.24 AS build
WORKDIR /src
COPY . .
# go mod tidy fills in go.sum on first build; commit go.sum afterwards
RUN go mod tidy && CGO_ENABLED=0 go build -o /api .

FROM gcr.io/distroless/static
COPY --from=build /api /api
EXPOSE 8080
ENTRYPOINT ["/api"]
