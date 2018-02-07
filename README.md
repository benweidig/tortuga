# Tortuga [![Build Status](https://travis-ci.org/benweidig/tortuga.svg?branch=master)](https://travis-ci.org/benweidig/tortuga)

CLI tool for fetching/pushing/rebasing multiple git repositories at once.

## Work in Progress

THIS TOOL IS A WORK IN PROGRESS!
NOT BATTLE-TESTED AT ALL!
TREAD WITH CARE!

As soon as I got a _"works reliably on my machine"_-version I will add releases.


## Usage
```
tortuga [-l/--local-only] [<path>]
```

## Arguments

| Argument          | Default | Description           |
| ----------------- | ------- | --------------------- |
| path              | .       | Path to repositories  |
| -l / --local-only | false   | Don't update remotes  |
| -m / --monochrome | false   | Don't use ANSI colors |


## License

MIT. See [LICENSE](LICENSE).
