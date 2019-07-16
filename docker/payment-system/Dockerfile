# Start from golang v1.12.7 base image
FROM golang:1.12.7-alpine as builder

# Add Maintainer Info
LABEL maintainer="Leandr Khaliullov <leandr@cpan.org>"

RUN apk add make git gcc libc-dev

# Set the Current Working Directory inside the container
WORKDIR /go/src/github.com/khaliullov/payment-system

# Copy everything from the current directory to the PWD(Present Working Directory) inside the container
COPY . .

# Download dependencies, build and install
RUN make build install

######## Start a new stage from scratch #######
FROM alpine:3.10.1

RUN apk --no-cache add ca-certificates curl

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /go/src/github.com/khaliullov/payment-system/payment-system /usr/bin/

CMD ["/usr/bin/payment-system"]
