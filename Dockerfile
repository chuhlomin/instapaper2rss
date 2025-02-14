FROM golang:1.22 AS build-env
WORKDIR /src/
COPY . /src/
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -ldflags="-w -s" -mod=vendor -o /go/bin/app /src/

FROM gcr.io/distroless/static:latest

LABEL name="render-template"
LABEL repository="http://github.com/chuhlomin/instapaper2rs"
LABEL homepage="http://github.com/chuhlomin/instapaper2rs"
LABEL maintainer="Constantine Chukhlomin <mail@chuhlomin.com>"
LABEL com.github.actions.name="Instapaper to RSS"
LABEL com.github.actions.description="Converts Instapaper bookmarks to RSS feed"
LABEL com.github.actions.icon="file-text"
LABEL com.github.actions.color="purple"

COPY --from=build-env /go/bin/app /app
CMD ["/app"]
