{ pkgs ? import <nixpkgs> {} }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    go
    gopls
    go-tools
    delve
    golangci-lint
    git
    gnumake
  ];

  shellHook = ''
    echo "Go development environment loaded"
    echo "Go version: $(go version)"
    echo ""
    echo "Available tools:"
    echo "  go        - Go compiler and toolchain"
    echo "  gopls     - Go language server"
    echo "  dlv       - Delve debugger"
    echo "  golangci-lint - Go linter"
    echo ""
  '';
}