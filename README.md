# tools
General Purpose Tools

## Install

```bash
$ go install github.com/jurteam/tools/...@latest
```

## storeProof

Set the environemt variable before launching the program `W3FS_API_KEY`:
```
$ export W3FS_API_KEY="HEREMYKEY"
```

Ensure that the `proof.json` file exists in your current working directory;
alternatively you can use the `-filename` flag:

```
$ storeProof
```
