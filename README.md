# esbuild-service

[esbuild](https://github.com/evanw/esbuild) as a service.

## Install

```bash
curl -sf https://gobinaries.com/egoist/esbuild-service | sh
```

Then `esbuild-service` command will be available.

The first time you send a request will be slow because it needs to install a few npm package to shim Node.js built-in modules.

## Development

```bash
# Start server
make run
# the open http://localhost:8080

# Build
make build
# then run `./esbuilds-service`
```

## Docs

### Environment variables

- `PORT`: Server port, defaults to `8080`.

### `/*pkg`

Build an npm package, `pkg` can be:

- A bare name like `vue`
- Name with version: `vue@3.0.0-rc.1`
- Name, version and a file path: `preact@10/compact`

Query parameters:

- `format`: Bundle format, defaults to `esm`, available values: `cjs`, `iife`
- `globalName`: Global variable name for `iife` bundle.

## License

MIT &copy; [EGOIST](https://github.com/sponsors/egoist)
