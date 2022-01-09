# opaq

`opaq` is a generic inquiry tool to OPA server. A major purpose of this tool is for inquiry in GitHub Actions.

## Features

- **Data formatting**: OPA server accepts only `{"input": ...}` schema and responds `{"result": ...}` schema. `opaq` changes input format and extracts result data before/after inquiry to OPA server.
- **Control exit code**: `--fail-defined` and `--fail-undefined` options can change exit code to fail CI.
- **Inject metadata**: `--metadata (-m)` can inject metadata to original input data for more sophisticated decision.

## Usage

Installation with `go` command.

```bash
$ go install github.com/m-mizutani/opaq@latest
```

Or run command via docker image `ghcr.io/m-mizutani/opaq:latest`.

```bash
$ docker run ghcr.io/m-mizutani/opaq:latest -i result.json -u https://your-opa-server/v1/data/yourpolicy
```

### Basic

```bash
$ opaq -i result.json -u https://your-opa-server/v1/data/yourpolicy
{
    "allow": true
}
```

### GitHub Actions

E.g. querying a result of [Trivy](https://github.com/aquasecurity/trivy) scan.

```yml
name: Vuln scan and inquiry to OPA server

on: [push]

jobs:
  scan:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout upstream repo
        uses: actions/checkout@v2
        with:
          ref: ${{ github.head_ref }}
      - name: Run Trivy vulnerability scanner in repo mode
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: fs
          format: json
          output: trivy-results.json
          list-all-pkgs: true
      - uses: docker://ghcr.io/m-mizutani/opaq:latest
        with:
          args: "-u https://your-opa-server/v1/data/trivy -i trivy-results.json -m repository=${{ github.repository }} -m ref=${{ github.ref_name }} --fail-defined"
```

### Control exit code

`opaq` has two options for non-zero code exit to fail CI.

- `--fail-defined`: Exits with non-zero exit code on **undefined/empty** result and errors
- `--fail-undefined`: Exits with non-zero exit code on **defined/non-empty** result and errors

```bash
$ opaq -i result.json -u https://your-opa-server/v1/data/blue --fail-defined
{
    "allow": true
}
# Exit with non-zero code
```

```bash
$ opaq -i result.json -u https://your-opa-server/v1/data/orange --fail-defined
{}
# Normally exit
```

### Inject metadata

In some cases, the structural data output for evaluation by OPA is not enough information for evaluation. For example, evaluation requires not only content of configuration file but also directory path and file name to check consistency. `opaq` allows to add metadata to original structure data.

```bash
$ opaq -i some/file.json -m "path=some/file.json" -u https://your-opa-server/v1/data/green
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

### Other options

- `--input`: Specify input file instead of STDIN
- `--format`: Choose input format [`json`, `yaml`]
- `--data-field`: Nest input data with a value of the option. If `mydata` is provided, `{"user":"you"}` will be modified to `{"mydata":{"user":"you"}}`
- `http-header`: Add custom HTTP header(s). e.g. `Authorization: Bearer XXXXX` to pass authentication of OPA server

## License

Apache License 2.0
