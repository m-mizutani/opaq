# opaq

`opaq` is a generic inquiry tool to OPA server. A major purpose of this tool is for inquiry in GitHub Actions.

## Features

- **Control exit code**: `--fail-defined` and `--fail-undefined` options can change exit code to fail CI.
- **Inject metadata**: `--metadata (-m)` can inject metadata to original input data for more sophisticated decision.

## Usage

### Basic

```bash
$ some-command | opaq -u https://your-opa-server/v1/data/yourpolicy
{
    "allow": true
}
```

### Control exit code

Two option to exit with non-zero code.

- `--fail-defined`: Exits with non-zero exit code on **undefined/empty** result and errors
- `--fail-undefined`: Exits with non-zero exit code on **defined/non-empty** result and errors

```bash
$ some-command | opaq -u https://your-opa-server/v1/data/blue --fail-defined
{
    "allow": true
}
# Exit with non-zero code
```

```bash
$ some-command | opaq -u https://your-opa-server/v1/data/orange --fail-defined
{}
# Normally exit
```

### Inject metadata

In some cases, the structural data output for evaluation by OPA is not enough information for evaluation. For example, evaluation requires not only content of configuration file but also directory path and file name to check consistency. `opaq` allows to add metadata to original structure data.

```bash
$ cat some/file.json | opaq -m "path=some/file.json" -u https://your-opa-server/v1/data/green
```

If original `some/file.json` is below,

```json
{
    "config": {...}
}
```

`-m` option modifies data as following and send it to OPA server.

```json
{
    "config": {...},
    "metadata": {
        "path": "some/file.json"
    }
}
```

Also, `--metadata-field` can change a field name of metadata. Default is `metadata`.

## License

Apache License 2.0
