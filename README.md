# protocw

A protoc wrapper that helps with dependency management when generating **Go**
files (`.pb.go`s) from protobufs.

Its job is to take a specification containing protos and the repos to find them
in, and build `.pb.go`s for all of them, placing the results into a vendor
directory.

## Caveat emptor

This tool is experimental. Please file issues for bugs or new use cases.

This tool ignores all `option go_package` declarations. It always rewrites
imports to something that works locally.

This tool only works in the context of a Go module aware program, as a
side-affect of the import rewriting.

This tool always vendors every single dependency. It is not capable of vendoring
a subset of the outputs.

## Usage

### Simple example

Given this simple proto file,

```
# github.com/my/repo/people.proto
message Person {
  string name = 1;
  int32 id = 2;
  string email = 3;
}
```

Create the dependency file,

```
# deps.yml
- repo: github.com/my/repo
  protos:
    - people.proto
```

Run,

```
go get -tool github.com/jeanbza/protocw
go tool github.com/jeanbza/protocw -c deps.yml
```

### More complex dependency

If `people.proto` depends on `import "foo/bar/gaz.proto"`, you'll have to figure
out which repo holds `foo/bar/gaz.proto` and include it in your config file:

```
# deps.yml
- repo: github.com/my/repo
  protos:
    - people.proto
- repo: github.com/some/lib
  protos:
    - foo/baz/gaz.proto
```

**All transitive dependencies** must be specified in your config file.
