# `update-go-tools`

`update-go-tools` is a lightweight utility designed to inspect, inventory, verify, and update Go developer binaries managed via `go install`. It leverages embedded module build metadata through `debug/buildinfo`.

## Command Usage

```
update-go-tools [tool...]
```

## Flags and Options

<table>
  <tr>
    <th>Flag</th>
    <th>Description</th>
  </tr>
  <tr>
    <td><code>--help</code>, <code>-h</code></td>
    <td>Display help documentation</td>
  </tr>
  <tr>
    <td><code>--version</code>, <code>-v</code></td>
    <td>Display version information</td>
  </tr>
  <tr>
    <td><code>--list</code></td>
    <td>Display inventory of installed Go tools with versions and modules</td>
  </tr>
  <tr>
    <td><code>--info &lt;tool&gt;</code></td>
    <td>Show detailed metadata for a specific tool</td>
  </tr>
  <tr>
    <td><code>--verify</code></td>
    <td>Verify integrity of installed Go tools without updating</td>
  </tr>
  <tr>
    <td><code>--check</code></td>
    <td>Preview updates without executing changes</td>
  </tr>
</table>

## Exit Codes

* 0 - Success
* 1 - Update failure
* 2 - Usage error
* 3 - Environment error
