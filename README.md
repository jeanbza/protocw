# protocw

A protoc wrapper that helps with dependency management when generating **Go**
files (`.pb.go`s) from protobufs.

Its job is to take a specification containing protos and the repos to find them
in, and build `.pb.go`s for all of them, replaces all their Go import statements
with statements that work locally, relative to your `go.mod`, and places the
results into a directory of your choosing (which should be inside your Go
module).

Its value is the ability to specify a dependency graph vertexes in a small file
that can then be exploded into `.pb.go`s. Under the hood it takes care of
finding protos in their repos, understanding the proto import paths to use,
understanding how to convert those into Go paths, understanding how to make
those Go paths work in your Go module, and passing the (often very extensive)
flags necessary for that to `protoc`.

## Caveat emptor

This tool is experimental. Please file issues for bugs or new use cases.

This tool ignores all `option go_package` declarations. It always rewrites
imports to something that works locally.

This tool only works in the context of a Go module aware program, as a
side-affect of the import rewriting.

This tool always vendors every single dependency. It is not capable of vendoring
a subset of the outputs.

## Usage

1. Install required tools `git`, `protoc`, `protoc-gen-go`, and optionally `protoc-gen-go-grpc`: https://grpc.io/docs/languages/go/quickstart/.
2. Download the tool: `go get -tool github.com/jeanbza/protocw`.
3. Enumerate protos in your dependency graph, and the repos containing them, in a yml file.
4. Run the tool: `go tool github.com/jeanbza/protocw -c deps.yml`. Optionally provide `-grpc` for grpc generation, too.

## Config

Dependency graphs are encoded in a yml file as a
`list[repo: string, protos: list[string]]`.

- Each repo is the git clone URL.
- Each proto is the _proto_ import path. For example, if proto `foo.proto` is
  imported by `bar.proto` as `import "a/b/foo.proto"`, then it should be
  referred to as `a/b/foo.proto` in the yml.

### Simple

Given this simple proto file,

```proto
# github.com/my/repo/people.proto
message Person {
  string name = 1;
  int32 id = 2;
  string email = 3;
}
```

Create the dependency file,

```yml
# deps.yml
- repo: github.com/my/repo
  protos:
    - people.proto
```

### More complex dependency

If `people.proto` depends on `import "foo/bar/gaz.proto"`, you'll have to figure
out which repo holds `foo/bar/gaz.proto` and include it in your config file:

```yml
# deps.yml
- repo: github.com/my/repo
  protos:
    - people.proto
- repo: github.com/some/lib
  protos:
    - foo/baz/gaz.proto
```

**All transitive dependencies** must be specified in your config file.
