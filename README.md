# esbuild-service

[esbuild](https://github.com/evanw/esbuild) as a service.

## Install

```bash
curl -sf https://gobinaries.com/egoist/esbuild-service | sh
```

Then `esbuild-service` command will be available.

## Development

```bash
# Start server
go run main.go
# the open http://localhost:8080

# Build
go build
# then run `./esbuilds-service`
```

## Docs

### Environment variables

- `PORT`: Server port, defaults to `8080`.

### `/build/*pkg`

Build an npm package, `pkg` can be:

- A bare name like `vue`
- Name with version: `vue@3.0.0-rc.1`
- Name, version and a file path: `preact/compact@10.0.0`

Query parameters:

- `format`: Bundle format, defaults to `esm`, available values: `cjs`, `iife`
- `globalName`: Global variable name for `iife` bundle.

## TODO

Support version range, e.g. `vue@^2` should automatically use the latest version that satifies the version range.

## License

MIT &copy; [EGOIST](https://github.com/sponsors/egoist)