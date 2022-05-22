{ pkgs ? import <nixpkgs> { } }:
pkgs.buildGoModule {
  pname = "authfish";
  version = "0.0.1";
  vendorSha256 = "sha256-2rqMtwKNa/U9pxbQgr+/PWn+4GkDoNNzW7At6XEXYaY=";
  src = ./.;
}
