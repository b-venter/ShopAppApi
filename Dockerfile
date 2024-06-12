FROM golang:1.21-alpine as build-stage

RUN apk --no-cache add ca-certificates
    
WORKDIR /home/avenger/shopapi

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -o /shopapi .
RUN apk add --no-cache --upgrade bash
#
# final build stage
#
FROM scratch

# Copy ca-certs for app web access
COPY --from=build-stage /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build-stage /shopapi /shopapi

# app uses port 4000
EXPOSE 4000

ENTRYPOINT ["/shopapi"] 
