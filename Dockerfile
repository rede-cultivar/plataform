# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags "-s -w \
      -X github.com/rede-cultivar/plataform/internal/version.Version=${VERSION} \
      -X github.com/rede-cultivar/plataform/internal/version.Commit=${COMMIT} \
      -X github.com/rede-cultivar/plataform/internal/version.Date=${BUILD_DATE}" \
    -o /out/cultivar ./cmd/cultivar

# Diretório usado pelo backend de armazenamento em disco (teste local). O
# volume nomeado herda a permissão da imagem, permitindo escrita pelo nonroot.
RUN mkdir -p /out/media && touch /out/media/.keep

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/cultivar /cultivar
COPY --chown=65532:65532 --from=build /out/media/ /data/media/

EXPOSE 8080

ENTRYPOINT ["/cultivar"]
CMD ["serve"]
