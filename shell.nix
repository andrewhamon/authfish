{ pkgs ? import <nixpkgs> { } }:
pkgs.mkShell {
  # nativeBuildInputs is usually what you want -- tools you need to run
  nativeBuildInputs = [
    pkgs.go
    pkgs.gopls
    pkgs.nixpkgs-fmt
    pkgs.delve
    pkgs.go-tools
  ];

  shellHook = ''
    export PATH=$PATH:$HOME/go/bin
  '';
}
